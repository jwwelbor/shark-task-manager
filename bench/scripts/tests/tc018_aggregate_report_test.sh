#!/usr/bin/env bash
# TC-018 (test-plan.md AC test matrix; T-E40-F03-003 task spec Test Cases),
# sub-cases a, b, c, d, e, f, q, s, t, w, x.
#
# T-E40-F03-003's slice: `aggregate-runs.sh`'s core only -- pinned-glob
# enumeration, per-record structural validation (AC-07), classification
# (AC-08/AC-09/AC-10), family presence read from the family block itself
# and never from `sources` (TC-018q, TD-076's consumer-side consequence),
# five-field provenance uniformity (AC-11/TC-018f), the pinned-glob-vs-find
# quarantine exclusion (TC-018s), and `batch-log.jsonl` non-read (TC-018t).
# TC-018w/TC-018x (post-UAT, uat-20260808-E40-F03.md F-4/F-7) close the
# "presence read as key-existence, not usable value" gap the first UAT
# round found in this same slice: a null (not just missing) family key at
# classify()/families_present (TC-018w), and a uniformity field's absence
# exemption scoped to the outcomes that actually explain it rather than a
# blanket skip (TC-018x).
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


def derivation_line(text):
    m = re.search(r"- Derivation:.*", text)
    assert m, "no Derivation line found: %r" % text
    return m.group(0)


def rule_values_from_derivation(text):
    # The printed derivation sentence must itself end with its own
    # evaluated "= [lo, hi]." -- extracted independently of
    # interval_from_report(), which reads the separate "Acceptance
    # interval:" line, so a test can assert the RULE (not merely the
    # aggregate own numbers) reproduces the published interval. F-9: the
    # r>0 sentence used to read "[min - r, max + r]" while the aggregator
    # actually applies "max(0, min - r)", so the printed rule and the
    # printed interval could disagree whenever the lower clamp bound.
    line = derivation_line(text)
    m = re.search(r"= \[([^,]+), ([^\]]+)\]\.$", line)
    assert m, "derivation sentence does not end with its own evaluated [lo, hi]: %r" % line
    return float(m.group(1)), float(m.group(2))


# (i) loc_prod_added: r > 0 branch -- interval independently recomputed
# from the aggregate own min/max, never copied from the implementation.
block = metrics["loc_prod_added"]
text = metric_block_text("loc_prod_added")
mn, mx = block["min"], block["max"]
r = mx - mn
expected_lo, expected_hi = max(0, mn - r), mx + r
lo, hi = interval_from_report(text)
assert abs(lo - expected_lo) < 1e-6 and abs(hi - expected_hi) < 1e-6, (lo, hi, expected_lo, expected_hi)
assert str(mn) in text and str(mx) in text and str(block["median"]) in text, text
assert "max(0" in text, "derivation sentence omits the lower-clamp branch the aggregator applies (F-9): %r" % text
rule_lo, rule_hi = rule_values_from_derivation(text)
assert rule_lo == lo and rule_hi == hi, (rule_lo, rule_hi, lo, hi)

# (ii) wall_clock_ns: r == 0, median != 0 branch.
block = metrics["wall_clock_ns"]
text = metric_block_text("wall_clock_ns")
assert block["spread_abs"] == 0 and block["median"] != 0, block
r_eff = 0.10 * abs(block["median"])
expected_lo, expected_hi = max(0, block["min"] - r_eff), block["max"] + r_eff
lo, hi = interval_from_report(text)
assert abs(lo - expected_lo) < 1e-3 and abs(hi - expected_hi) < 1e-3, (lo, hi, expected_lo, expected_hi)
assert "10%" in text, "no 10%%-of-median derivation sentence for the zero-spread/nonzero-median branch: %r" % text
assert "max(0" in text, "derivation sentence omits the lower-clamp branch the aggregator applies (F-9): %r" % text
rule_lo, rule_hi = rule_values_from_derivation(text)
assert abs(rule_lo - lo) < 1e-3 and abs(rule_hi - hi) < 1e-3, (rule_lo, rule_hi, lo, hi)

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

	# F-9 regression (uat-20260808-E40-F03.md): a Class C metric whose
	# lower clamp at 0 actually BINDS -- the reported repro was
	# min=0, max=10, r=10, where the aggregator's accept_lo =
	# max(0, min - r) = 0 but the old unclamped derivation sentence read
	# "interval = [min - r, max + r]" = [-10, 20], disagreeing with the
	# published [0, 20]. build_ac12_aggregate's own loc_prod_added fixture
	# (10, 12, 15) never binds the clamp, so this needs its own fixture.
	local clamp_root="$WORKDIR/o-clamp-root"
	local clamp_aggregate="$WORKDIR/o-clamp-aggregate.json"
	local clamp_item="f03-fixture-tc018o-clamp"
	local rep prod_added
	for rep in 1 2 3; do
		case "$rep" in
		1) prod_added=0 ;;
		2) prod_added=5 ;;
		3) prod_added=10 ;;
		esac
		place_record "$clamp_root" "$clamp_item" "$rep" --set "loc.prod_added=$prod_added"
	done

	local cout cerr ccode
	{
		read -r cout
		read -r cerr
		read -r ccode
	} < <(run_aggregate "$clamp_root")
	[[ "$ccode" -eq 0 ]] || fail "o(clamp): aggregate-runs.sh exited $ccode, want 0: $(cat "$cerr")"
	cp "$cout" "$clamp_aggregate"

	local rout rerr rcode
	{
		read -r rout
		read -r rerr
		read -r rcode
	} < <(run_report "$clamp_aggregate")
	[[ "$rcode" -eq 0 ]] || fail "o(clamp): report-baseline.sh exited $rcode, want 0: $(cat "$rerr")"

	python3 -c '
import json
import re
import sys

aggregate_path, report_path, item = sys.argv[1], sys.argv[2], sys.argv[3]
agg = json.load(open(aggregate_path))
report_text = open(report_path).read()

task = next(t for t in agg["tasks"] if t["item_id"] == item)
block = task["metrics"]["loc_prod_added"]
mn, mx = block["min"], block["max"]
assert mn == 0 and mx == 10, "fixture drifted, want min=0 max=10: %r" % block

marker = "#### `loc_prod_added`"
start = report_text.index(marker)
text = report_text[start:]
nxt = re.search(r"\n#### ", text[1:])
if nxt:
    text = text[: nxt.start() + 1]

m = re.search(r"Acceptance interval: \[([^,]+), ([^\]]+)\]", text)
assert m, "no Acceptance interval line: %r" % text
printed_lo, printed_hi = float(m.group(1)), float(m.group(2))
assert printed_lo == 0.0, "aggregate accept_lo not clamped as expected (fixture no longer reproduces F-9): %r" % block

dm = re.search(r"- Derivation:.*", text)
assert dm, "no Derivation line: %r" % text
derivation_line = dm.group(0)
assert "max(0" in derivation_line, (
    "derivation sentence omits the lower-clamp branch the aggregator "
    "applied -- F-9 regression: %r" % derivation_line
)

fm = re.search(r"= \[([^,]+), ([^\]]+)\]\.$", derivation_line)
assert fm, "derivation sentence does not end with its own evaluated [lo, hi]: %r" % derivation_line
rule_lo, rule_hi = float(fm.group(1)), float(fm.group(2))
assert rule_lo == printed_lo and rule_hi == printed_hi, (
    "printed derivation rule does not reproduce the printed Acceptance "
    "interval: %r" % ((rule_lo, rule_hi, printed_lo, printed_hi),)
)
r = mx - mn
assert rule_lo == max(0, mn - r) and rule_hi == mx + r, (rule_lo, rule_hi, mn, mx, r)
' "$clamp_aggregate" "$rout" "$clamp_item" || fail "o(clamp): assertion failed: $(cat "$rout")"

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

# ---------------------------------------------------------------------------
# TC-018w (UAT F-4, AC-08/REQ-F-008/REQ-F-009): a family key that EXISTS but
# holds JSON `null` must not be classified `complete` -- classify() must
# treat null exactly like a missing key (get_leaf already does this at the
# leaf level; this closes the same gap at the family level). Sub-case (a):
# an unexplained null family reaches the anomaly bucket and a non-zero
# exit, same as test_c's missing-key case. Sub-case (b), negative: the same
# null family alongside an explanation (toolchain_guard abort) still
# classifies explained_absence, not anomaly -- null-handling must not
# override the explanation check. Both sub-cases also close the sibling
# F-4 instance in `inventory[].families_present` (line ~685): a null
# family must never be listed as present there either.
# ---------------------------------------------------------------------------
test_w() {
	# --- (a) unexplained null family -> anomaly, non-zero exit. ---
	local root_a="$WORKDIR/w-root-a"
	place_record "$root_a" f03-fixture-tc018w-null 1 --set oracle=null

	local out_a err_a code_a
	{
		read -r out_a
		read -r err_a
		read -r code_a
	} < <(run_aggregate "$root_a")
	[[ "$code_a" -ne 0 ]] || fail "w(a): exited 0, want non-zero (null oracle family is an unexplained absence)"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
run_key = "f03-fixture-tc018w-null::default::rep1"
inv = d["inventory"][run_key]
assert inv["classification"] == "anomaly", inv
assert "oracle" not in inv["families_present"], "null oracle must not be listed as present: %r" % inv
assert "quality" in inv["families_present"], inv
assert "loc" in inv["families_present"], inv
matches = [a for a in d["anomalies"] if a["run_key"] == run_key]
assert len(matches) == 1, matches
assert matches[0]["missing_families"] == ["oracle"], matches[0]
assert d["outcomes"]["anomaly_count"] == 1, d["outcomes"]
' "$out_a" || fail "w(a): assertion failed: $(cat "$out_a")"
	echo "TC-018w(a, null family classifies anomaly, not complete) PASS"

	# --- (b) negative: null family WITH an explanation -> explained_absence. ---
	local root_b="$WORKDIR/w-root-b"
	place_record "$root_b" f03-fixture-tc018w-null-explained 1 \
		--set 'quality.toolchain_guard=go_version_mismatch' \
		--unset quality.fmt_clean --unset quality.vet_ok --unset quality.tests_pass \
		--unset quality.lint_new_issues --unset quality.lint_new_issues_count \
		--set oracle=null --unset loc \
		--unset sources.oracle --unset sources.loc \
		--set 'errors=[{"kind":"postrun_check_aborted","detail":"go version mismatch (toolchain guard abort)"}]'

	local out_b err_b code_b
	{
		read -r out_b
		read -r err_b
		read -r code_b
	} < <(run_aggregate "$root_b")
	[[ "$code_b" -eq 0 ]] || fail "w(b): exited $code_b, want 0 (null oracle is explained by toolchain_guard abort): $(cat "$err_b")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
run_key = "f03-fixture-tc018w-null-explained::default::rep1"
inv = d["inventory"][run_key]
assert inv["classification"] == "explained_absence", inv
assert "oracle" not in inv["families_present"], "null oracle must not be listed as present: %r" % inv
assert "quality" in inv["families_present"], inv
assert d["outcomes"]["anomaly_count"] == 0, d["outcomes"]
' "$out_b" || fail "w(b): assertion failed: $(cat "$out_b")"
	echo "TC-018w(b, null family with explanation is explained_absence, not anomaly) PASS"

	echo "TC-018w PASS"
}

# ---------------------------------------------------------------------------
# TC-018x (UAT F-7, REQ-F-011): provenance-uniformity absence scoping.
# Sub-case (a), positive/negative-control: a uniformity field absent for a
# reason its record's outcome explains (manifest.model_ids missing on a
# timeout record, alongside a completed record that carries it) is exempt
# -- still reported uniform, and the surviving present value publishes.
# Sub-case (b): the same absence on a `complete` record whose outcome does
# NOT explain it (manifest.fixture_base_sha simply unset) is NOT exempt --
# it must make the batch non-uniform, naming a `null` value alongside the
# sibling record's real value, and must never let a baseline_id be built
# from the other record's SHA alone. Sub-case (c) (rewritten for R2-F-11,
# round-2 UAT/round-3 code-review -- the original assertion here encoded
# the defect as intended behavior and was flagged for rewrite, not
# extension): the same illegitimate absence shared by EVERY contributing
# record is NOT verified agreement -- shared absence is not shared
# knowledge. It must make the batch non-uniform, name an explicit
# `unpinned_field` divergence, and suppress tasks[]/corpus/flags/
# baseline_id the same way any other non-uniform batch does.
# ---------------------------------------------------------------------------
test_x() {
	# --- (a) legitimate exemption: model_ids absent on a timeout record. ---
	local root_a="$WORKDIR/x-root-a"
	place_record "$root_a" f03-fixture-tc018x-legit-a 1
	place_record "$root_a" f03-fixture-tc018x-legit-b 1 --golden timeout

	local out_a err_a code_a
	{
		read -r out_a
		read -r err_a
		read -r code_a
	} < <(run_aggregate "$root_a")
	[[ "$code_a" -eq 0 ]] || fail "x(a): exited $code_a, want 0 (model_ids absence explained by timeout outcome): $(cat "$err_a")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
prov = d["provenance"]
assert prov["uniform"] is True, prov
assert "divergences" not in prov, prov
assert prov.get("model_ids") == ["claude-sonnet-5"], prov
' "$out_a" || fail "x(a): assertion failed: $(cat "$out_a")"
	echo "TC-018x(a, model_ids absent on a timeout record is exempt) PASS"

	# --- (b) illegitimate absence: fixture_base_sha unset on a complete record. ---
	local root_b="$WORKDIR/x-root-b"
	place_record "$root_b" f03-fixture-tc018x-illegit-a 1
	place_record "$root_b" f03-fixture-tc018x-illegit-b 1 --unset manifest.fixture_base_sha

	local out_b err_b code_b
	{
		read -r out_b
		read -r err_b
		read -r code_b
	} < <(run_aggregate "$root_b")
	[[ "$code_b" -ne 0 ]] || fail "x(b): exited 0, want non-zero (fixture_base_sha absent on a complete record is a divergence)"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
prov = d["provenance"]
assert prov["uniform"] is False, prov
divs = prov.get("divergences", [])
matches = [dv for dv in divs if dv["field"] == "fixture_base_sha"]
assert len(matches) == 1, "expected exactly one fixture_base_sha divergence, got %r" % divs
values = {json.dumps(v["value"], sort_keys=True) for v in matches[0]["values"]}
assert "null" in values, "missing field must surface as an explicit null value: %r" % matches[0]
assert len(matches[0]["values"]) == 2, matches[0]
assert "fixture_base_sha" not in prov, "divergent field must not also appear as a single agreed value: %r" % prov
assert "baseline_id" not in d, "non-uniform provenance must never publish baseline_id: %r" % d.get("baseline_id")
' "$out_b" || fail "x(b): assertion failed: $(cat "$out_b")"
	echo "TC-018x(b, fixture_base_sha absent on a complete record is a divergence, not silently skipped) PASS"

	# --- (c) illegitimate absence shared by every contributing record: NOT
	# verified agreement -- shared absence, not shared knowledge (R2-F-11).
	local root_c="$WORKDIR/x-root-c"
	place_record "$root_c" f03-fixture-tc018x-allmissing-a 1 --unset manifest.fixture_base_sha
	place_record "$root_c" f03-fixture-tc018x-allmissing-b 1 --unset manifest.fixture_base_sha

	local out_c err_c code_c
	{
		read -r out_c
		read -r err_c
		read -r code_c
	} < <(run_aggregate "$root_c")
	[[ "$code_c" -ne 0 ]] || fail "x(c): exited 0, want non-zero (every record sharing an unexplained absence is NOT verified pinning)"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
prov = d["provenance"]
assert prov["uniform"] is False, prov
assert "fixture_base_sha" not in prov, "an unpinned field must not also appear as a single agreed value: %r" % prov
divs = prov.get("divergences", [])
matches = [dv for dv in divs if dv["field"] == "fixture_base_sha"]
assert len(matches) == 1, "expected exactly one fixture_base_sha divergence, got %r" % divs
assert matches[0].get("reason") == "unpinned_field", matches[0]
values = {json.dumps(v["value"], sort_keys=True) for v in matches[0]["values"]}
assert values == {"null"}, "shared absence must name the null value every record shares, not fabricate a pair: %r" % matches[0]
assert "tasks" not in d, "non-uniform provenance must never publish tasks[]: %r" % d.get("tasks")
assert "corpus" not in d, "non-uniform provenance must never publish corpus: %r" % d.get("corpus")
assert "flags" not in d, "non-uniform provenance must never publish flags: %r" % d.get("flags")
assert "baseline_id" not in d, "non-uniform provenance must never publish baseline_id: %r" % d.get("baseline_id")
' "$out_c" || fail "x(c): assertion failed: $(cat "$out_c")"
	echo "TC-018x(c, absence shared by every record is unpinned, not agreed -- non-uniform) PASS"

	echo "TC-018x PASS"
}

# ---------------------------------------------------------------------------
# TC-018y (post-UAT round 1, uat-20260808-E40-F03.md F-3: REQ-N-005/
# REQ-F-012/REQ-F-016 -- "a band is never published over a silently
# reduced rep set"): a matrix hole. item-a has all 3 reps; item-b has
# only reps 1-2 (rep 3 never ran -- no record.jsonl at all, not an
# excluded/timeout/anomaly record; the r3 count is entirely driven by
# item-a). The header must still publish reps=3 (the declared/observed
# matrix size), but item-b's own bands must carry an explicit
# `missing_run` excluded[] entry for the hole and item-b must be named
# in a reduced-item flag -- never an empty excluded[] alongside a
# silently smaller n.
# ---------------------------------------------------------------------------
test_y() {
	local root="$WORKDIR/y-root"
	local item_a="f03-fixture-tc018y-a" item_b="f03-fixture-tc018y-b"
	local rep
	for rep in 1 2 3; do
		place_record "$root" "$item_a" "$rep"
	done
	for rep in 1 2; do
		place_record "$root" "$item_b" "$rep"
	done

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -eq 0 ]] || fail "y: exited $code, want 0 (a matrix hole is not itself an anomalous record): $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item_a, item_b = sys.argv[2], sys.argv[3]

prov = d["provenance"]
assert prov["reps"] == 3, prov
assert d["baseline_id"].endswith("-r3"), d["baseline_id"]

task_a = next(t for t in d["tasks"] if t["item_id"] == item_a)
task_b = next(t for t in d["tasks"] if t["item_id"] == item_b)

missing_key = item_b + "::default::rep3"

# item-a is unaffected: full n=3, no missing_run anywhere. (quality_tests_pass
# is excluded on every rep for an unrelated, pre-existing reason -- the
# golden own quality.tests_pass is null, gate_not_executed -- same carve-out
# TC-018h uses.)
for metric_id, block in task_a["metrics"].items():
    assert not any(e["reason"] == "missing_run" for e in block["excluded"]), block["excluded"]
    if metric_id == "quality_tests_pass":
        continue
    assert block["n"] == 3, "%s: n=%r, want 3 (item-a has no hole)" % (metric_id, block["n"])

# item-b: every metric must carry n=2 (never a silently inflated n) AND
# a missing_run excluded[] entry naming the absent rep -- the exact
# defect: "the header says r3, the band says n=2, and nothing connects
# them" must no longer be true.
saw_missing_run = 0
for metric_id, block in task_b["metrics"].items():
    reasons = {e["run_key"]: e["reason"] for e in block["excluded"]}
    assert reasons.get(missing_key) == "missing_run", "%s: excluded[] missing a missing_run entry for %s: %r" % (metric_id, missing_key, block["excluded"])
    saw_missing_run += 1
    if metric_id == "quality_tests_pass":
        continue  # n=0 for an unrelated, pre-existing reason (see item-a note above)
    assert block["n"] == 2, "%s: n=%r, want 2" % (metric_id, block["n"])
assert saw_missing_run > 5, "too few metrics carried the missing_run entry to be a meaningful check: %d" % saw_missing_run

# The reduced item must be named in a flag connecting r3 (published) to
# n=2 (contributed) -- REQ-N-005s own "nothing connects them" gap.
reduced = {r["item_id"]: r for r in d["flags"]["reduced_reps"]}
assert item_b in reduced, d["flags"]["reduced_reps"]
assert reduced[item_b]["contributing_reps"] == 2, reduced[item_b]
assert reduced[item_b]["published_reps"] == 3, reduced[item_b]
assert item_a not in reduced, d["flags"]["reduced_reps"]
' "$out" "$item_a" "$item_b" || fail "y: assertion failed: $(cat "$out")"
	echo "TC-018y(a: implicit matrix derivation from observed reps) PASS"

	# --- (b) explicit --reps: a hole even the OBSERVED matrix can't reveal
	# -- every item only ran 2 reps, but the declared matrix is 3 (rep 3
	# never ran for ANYONE). Without an explicit --reps argument there is
	# no way to know the matrix was supposed to be bigger than what
	# survived -- this is exactly why the fix note requires deriving the
	# expected count from the declared matrix/--reps, not purely from
	# surviving artifacts. ---
	local root_b="$WORKDIR/y-root-b"
	local item_c="f03-fixture-tc018y-c"
	for rep in 1 2; do
		place_record "$root_b" "$item_c" "$rep"
	done

	local out2 err2 code2
	{
		read -r out2
		read -r err2
		read -r code2
	} < <(run_aggregate "$root_b" --reps 3)
	[[ "$code2" -eq 0 ]] || fail "y(b): exited $code2, want 0: $(cat "$err2")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item_c = sys.argv[2]
prov = d["provenance"]
assert prov["reps"] == 3, prov

task_c = next(t for t in d["tasks"] if t["item_id"] == item_c)
missing_key = item_c + "::default::rep3"
for metric_id, block in task_c["metrics"].items():
    reasons = {e["run_key"]: e["reason"] for e in block["excluded"]}
    assert reasons.get(missing_key) == "missing_run", "%s: excluded[] missing a missing_run entry for %s: %r" % (metric_id, missing_key, block["excluded"])
    if metric_id == "quality_tests_pass":
        continue
    assert block["n"] == 2, "%s: n=%r, want 2" % (metric_id, block["n"])

reduced = {r["item_id"]: r for r in d["flags"]["reduced_reps"]}
assert item_c in reduced, d["flags"]["reduced_reps"]
assert reduced[item_c]["contributing_reps"] == 2, reduced[item_c]
assert reduced[item_c]["published_reps"] == 3, reduced[item_c]
' "$out2" "$item_c" || fail "y(b): assertion failed: $(cat "$out2")"
	echo "TC-018y(b: explicit --reps reveals a hole no surviving artifact could) PASS"

	echo "TC-018y PASS"
}

# ---------------------------------------------------------------------------
# TC-018z (post-UAT round 1, uat-20260808-E40-F03.md F-8: REQ-F-010/
# REQ-N-005/AC-12 internal consistency -- Class A counting and interval
# derivation must use ONE truthiness rule, and no metric's value type may
# reach class-specific arithmetic unvalidated). Two reps of
# `oracle.f2p_resolved` set to the JSON STRING "false" (not the JSON
# boolean) -- `v is True` (old true_count rule) says 0/2, `bool(v)` (old
# accept_set rule) says every rep truthy: exactly the F-8 contradiction.
# Fixed behavior: a non-boolean Class A leaf is excluded (never coerced),
# so a malformed record contributes nothing rather than a fabricated
# measurement.
# ---------------------------------------------------------------------------
test_z() {
	local root="$WORKDIR/z-root"
	local item="f03-fixture-tc018z"
	place_record "$root" "$item" 1 --set 'oracle.f2p_resolved="false"'
	place_record "$root" "$item" 2 --set 'oracle.f2p_resolved="false"'
	place_record "$root" "$item" 3 --set 'oracle.f2p_resolved=true'

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -eq 0 ]] || fail "z: exited $code, want 0: $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item = sys.argv[2]
task = next(t for t in d["tasks"] if t["item_id"] == item)
block = task["metrics"]["oracle_f2p_resolved"]

# Only rep3 (the real boolean) contributes -- n=1, insufficient_reps, no
# accept_set (n<2), and definitely no rate/accept_set contradiction.
assert block["n"] == 1, block
assert block.get("insufficient_reps") is True, block
assert "accept_set" not in block, block
assert block.get("true_count") == 1, block
assert block.get("rate") == 1.0, block

reasons = {e["run_key"]: e["reason"] for e in block["excluded"]}
assert reasons.get(item + "::default::rep1") == "invalid_value_type", reasons
assert reasons.get(item + "::default::rep2") == "invalid_value_type", reasons

# The item must NOT be flagged non_discriminative -- n<2 real
# contributions is vacuous, not evidence of an identical-across-reps
# result (mirrors the existing >=2-reps rule for genuine booleans).
assert item not in d["flags"]["non_discriminative_tasks"], d["flags"]["non_discriminative_tasks"]
assert task["non_discriminative"] is False, task
' "$out" "$item" || fail "z: assertion failed: $(cat "$out")"

	# --- Negative: two GENUINE booleans, both false -- must still exclude
	# nothing and still flag non_discriminative, proving the fix does not
	# over-exclude real boolean values. ---
	local root2="$WORKDIR/z-root2"
	place_record "$root2" "$item" 1 --set 'oracle.f2p_resolved=false'
	place_record "$root2" "$item" 2 --set 'oracle.f2p_resolved=false'

	local out2 err2 code2
	{
		read -r out2
		read -r err2
		read -r code2
	} < <(run_aggregate "$root2")
	[[ "$code2" -eq 0 ]] || fail "z(negative): exited $code2, want 0: $(cat "$err2")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item = sys.argv[2]
task = next(t for t in d["tasks"] if t["item_id"] == item)
block = task["metrics"]["oracle_f2p_resolved"]
assert block["n"] == 2, block
assert block["excluded"] == [], block["excluded"]
assert block["true_count"] == 0, block
assert block["rate"] == 0.0, block
assert block["accept_set"] == [False], block
assert item in d["flags"]["non_discriminative_tasks"], d["flags"]["non_discriminative_tasks"]
' "$out2" "$item" || fail "z(negative): assertion failed: $(cat "$out2")"

	echo "TC-018z PASS"
}

# ---------------------------------------------------------------------------
# TC-018aa (F-8 sweep site #1): a Class C `sum_usage` metric whose
# contributing stage's usage sub-field is a JSON string, not a number
# (`stages[0].usage.input_tokens` set to `"1000"`). Same defect class as
# F-8 -- a value read from the record reaches `total +=` arithmetic
# without a type check. Must exclude with `invalid_value_type`, never
# crash and never silently coerce/concatenate.
# ---------------------------------------------------------------------------
test_aa() {
	local root="$WORKDIR/aa-root"
	local item="f03-fixture-tc018aa"
	place_record "$root" "$item" 1 --set 'stages[0].usage.input_tokens="1000"'
	place_record "$root" "$item" 2
	place_record "$root" "$item" 3

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -eq 0 ]] || fail "aa: exited $code, want 0 (a malformed metric value excludes, never crashes the whole aggregation): $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item = sys.argv[2]
task = next(t for t in d["tasks"] if t["item_id"] == item)
block = task["metrics"]["tokens_input_total"]
assert block["n"] == 2, block
reasons = {e["run_key"]: e["reason"] for e in block["excluded"]}
assert reasons.get(item + "::default::rep1") == "invalid_value_type", reasons
' "$out" "$item" || fail "aa: assertion failed: $(cat "$out")"

	echo "TC-018aa PASS"
}

# ---------------------------------------------------------------------------
# TC-018bb (F-8 sweep site #2): `rejections.by_gate.<gate>` holding a JSON
# string instead of an integer. Same defect class -- `bg.get(gate, 0)` is
# handed straight to `min`/`max`/`mean` with no type check. Must exclude
# with `invalid_value_type`.
# ---------------------------------------------------------------------------
test_bb() {
	local root="$WORKDIR/bb-root"
	local item="f03-fixture-tc018bb"
	place_record "$root" "$item" 1 --set 'rejections.by_gate={"in_qa": "2"}'
	place_record "$root" "$item" 2
	place_record "$root" "$item" 3

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -eq 0 ]] || fail "bb: exited $code, want 0: $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item = sys.argv[2]
task = next(t for t in d["tasks"] if t["item_id"] == item)
block = task["metrics"]["rejections_by_gate.in_qa"]
reasons = {e["run_key"]: e["reason"] for e in block["excluded"]}
assert reasons.get(item + "::default::rep1") == "invalid_value_type", reasons
assert block["n"] == 2, block
' "$out" "$item" || fail "bb: assertion failed: $(cat "$out")"

	echo "TC-018bb PASS"
}

# ---------------------------------------------------------------------------
# TC-018cc (UAT round 2, R2-F-4, aggregate side: aggregate-runs.sh:841-842
# "if not distinct: continue # never observed anywhere"): a uniformity
# field legitimately absent on EVERY contributing record (model_ids on an
# all-timeout batch -- field_absence_explained() exempts every record, so
# field_values["model_ids"] never gets a single entry) must NOT vanish
# from provenance{} without a trace. It must publish as an explicit
# `provenance["model_ids"] = None` plus a `provenance["unresolved_fields"]`
# entry naming the reason ("all-exempt": every contributing record's
# absence was individually explained), distinct from F-7/x(c)'s
# "illegitimate absence shared by every record" case (which already
# publishes `null` via the normal single-distinct-value path and needs no
# unresolved_fields entry).
# ---------------------------------------------------------------------------
test_cc() {
	local root="$WORKDIR/cc-root"
	local item_a="f03-fixture-tc018cc-a" item_b="f03-fixture-tc018cc-b"
	place_record "$root" "$item_a" 1 --golden timeout
	place_record "$root" "$item_b" 1 --golden timeout

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -eq 0 ]] || fail "cc: exited $code, want 0 (model_ids absence explained on every record): $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
prov = d["provenance"]
assert prov["uniform"] is True, prov
assert "divergences" not in prov, prov
assert "model_ids" in prov and prov["model_ids"] is None, "an all-exempt field must publish as an explicit null, not be dropped: %r" % prov
unresolved = prov.get("unresolved_fields", [])
matches = [u for u in unresolved if u["field"] == "model_ids"]
assert len(matches) == 1, "expected exactly one model_ids unresolved_fields entry, got %r" % unresolved
assert matches[0]["reason"] == "all-exempt", matches[0]
' "$out" || fail "cc: assertion failed: $(cat "$out")"

	echo "TC-018cc PASS"
}

# ---------------------------------------------------------------------------
# TC-018dd (post-UAT round 2, uat-20260808-231500-E40-F03.md R2-F-7:
# REQ-N-005/REQ-F-012/REQ-F-016 -- "expected_rep_set must be authoritative
# in BOTH directions"): item-d has 3 real reps on disk, but the caller
# declares --reps 2. Rep 3 must never silently contribute to item-d's band
# while provenance.reps/baseline_id still say 2 (TC-018y only ever covers a
# matrix hole -- a rep count SMALLER than declared -- never a rep count
# LARGER than declared).
# ---------------------------------------------------------------------------
test_dd() {
	local root="$WORKDIR/dd-root"
	local item_d="f03-fixture-tc018dd-d"
	local rep
	for rep in 1 2 3; do
		place_record "$root" "$item_d" "$rep"
	done

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root" --reps 2)
	[[ "$code" -eq 0 ]] || fail "dd: exited $code, want 0 (an extra rep is flagged, not fatal): $(cat "$err")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item_d = sys.argv[2]

prov = d["provenance"]
assert prov["reps"] == 2, prov
assert d["baseline_id"].endswith("-r2"), d["baseline_id"]

task_d = next(t for t in d["tasks"] if t["item_id"] == item_d)
unexpected_key = item_d + "::default::rep3"

# Every metric must carry an unexpected_rep excluded[] entry for rep 3 AND
# n must never exceed 2 -- the exact defect: rep 3 silently inflating the
# band while the header says r2.
saw_unexpected = 0
for metric_id, block in task_d["metrics"].items():
    reasons = {e["run_key"]: e["reason"] for e in block["excluded"]}
    assert reasons.get(unexpected_key) == "unexpected_rep", "%s: excluded[] missing an unexpected_rep entry for %s: %r" % (metric_id, unexpected_key, block["excluded"])
    saw_unexpected += 1
    if metric_id == "quality_tests_pass":
        continue  # n=0 for an unrelated, pre-existing reason (see TC-018y note)
    assert block["n"] == 2, "%s: n=%r, want 2 (rep 3 must never contribute)" % (metric_id, block["n"])
assert saw_unexpected > 5, "too few metrics carried the unexpected_rep entry to be a meaningful check: %d" % saw_unexpected

# The item must be named in a flag connecting r2 (published) to the extra
# rep actually observed -- same discipline as flags.reduced_reps, mirrored
# for the opposite-direction mismatch.
unexpected = {r["item_id"]: r for r in d["flags"]["unexpected_reps"]}
assert item_d in unexpected, d["flags"]["unexpected_reps"]
assert unexpected[item_d]["unexpected_reps"] == [3], unexpected[item_d]
assert unexpected[item_d]["published_reps"] == 2, unexpected[item_d]

# item-d must NOT also appear in reduced_reps -- it has every declared rep
# (1, 2) plus an extra one; it is not missing anything.
reduced = {r["item_id"]: r for r in d["flags"]["reduced_reps"]}
assert item_d not in reduced, d["flags"]["reduced_reps"]
' "$out" "$item_d" || fail "dd: assertion failed: $(cat "$out")"

	echo "TC-018dd PASS"
}

# ---------------------------------------------------------------------------
# TC-018ee (UAT round 2, R2-F-2, report-baseline.sh side: REQ-F-011/
# REQ-F-022/AC-11/AC-21/REQ-N-005): a non-uniform aggregate (model_ids
# diverges across two contributing records, the AC-11 fixture shape) must
# render an unmissable "INVALID AS A BASELINE" banner, a divergences table
# naming the field/both values/run_keys, the diverging field's own
# Provenance row rendered as "_absent from the aggregate_" rather than
# silently dropped (fix guidance #2 -- the same `if key not in provenance:
# continue` mechanism R2-F-4/TC-018hh closes for the unresolved case), and
# the report itself must exit non-zero -- never a clean-looking "Shark Bench
# Baseline Report" over a batch the aggregator declared invalid.
# ---------------------------------------------------------------------------
test_ee() {
	local root="$WORKDIR/ee-root"
	local item="f03-fixture-tc018ee"
	place_record "$root" "$item" 1
	place_record "$root" "$item" 2 --set 'manifest.model_ids=["claude-evil-9"]'

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -ne 0 ]] || fail "ee: aggregate-runs.sh exited 0, want non-zero (non-uniform provenance)"

	local rout rerr rcode
	{
		read -r rout
		read -r rerr
		read -r rcode
	} < <(run_report "$out")
	[[ "$rcode" -ne 0 ]] || fail "ee: report-baseline.sh exited 0, want non-zero (a non-uniform batch must never be published as a clean baseline)"
	[[ -s "$rout" ]] || fail "ee: report-baseline.sh wrote nothing to stdout -- the invalid-baseline document must still print (naming the divergence), not disappear silently"

	local report_text
	report_text="$(cat "$rout")"
	grep -qi "INVALID AS A BASELINE" <<<"$report_text" || fail "ee: no INVALID AS A BASELINE banner in report: $report_text"
	grep -qi "model_ids\|Model IDs" <<<"$report_text" || fail "ee: divergent field 'model_ids' not named in report: $report_text"
	grep -q "claude-evil-9" <<<"$report_text" || fail "ee: diverging value not shown in report: $report_text"
	grep -q "${item}::default::rep1" <<<"$report_text" || fail "ee: divergence table does not name run_key rep1: $report_text"
	grep -q "${item}::default::rep2" <<<"$report_text" || fail "ee: divergence table does not name run_key rep2: $report_text"
	grep -qi "_absent from the aggregate_" <<<"$report_text" || fail "ee: diverging field's own Provenance row is not rendered as absent (R2-F-2 fix guidance #2): $report_text"

	echo "TC-018ee PASS"
}

# ---------------------------------------------------------------------------
# TC-018ff (UAT round 2, R2-F-3(a), report-baseline.sh side: REQ-N-005/
# REQ-F-012/REQ-F-021): the round-1 F-3 fixture verbatim (item-a reps 1-3,
# item-b reps 1-2, TC-018y's own fixture) rendered through report-
# baseline.sh must name item-b's rep reduction in a new "Data quality"
# section (contributing-vs-published rep counts), never leave the header's
# "Reps: 3" and the band's "n=2" unconnected -- round 1's own complaint,
# reproduced one layer downstream in the human-facing publication. A
# reduced rep set alone (no anomaly) is not itself invalid, so the report
# still exits 0.
# ---------------------------------------------------------------------------
test_ff() {
	local root="$WORKDIR/ff-root"
	local item_a="f03-fixture-tc018ff-a" item_b="f03-fixture-tc018ff-b"
	local rep
	for rep in 1 2 3; do
		place_record "$root" "$item_a" "$rep"
	done
	for rep in 1 2; do
		place_record "$root" "$item_b" "$rep"
	done

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -eq 0 ]] || fail "ff: aggregate-runs.sh exited $code, want 0: $(cat "$err")"

	local rout rerr rcode
	{
		read -r rout
		read -r rerr
		read -r rcode
	} < <(run_report "$out")
	[[ "$rcode" -eq 0 ]] || fail "ff: report-baseline.sh exited $rcode, want 0 (a reduced rep set alone is not an anomaly): $(cat "$rerr")"

	python3 -c '
import json
import sys

aggregate_path, report_path, item_a, item_b = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4]
agg = json.load(open(aggregate_path))
report_text = open(report_path).read()

reduced = {r["item_id"]: r for r in agg["flags"]["reduced_reps"]}
assert item_b in reduced, agg["flags"]["reduced_reps"]
entry = reduced[item_b]

start = report_text.index("## Data quality")
end = report_text.index("## Noise band per task")
section = report_text[start:end]

assert item_b in section, "reduced item %s not named in Data quality section: %r" % (item_b, section)
assert str(entry["contributing_reps"]) in section, "contributing_reps (%d) not named in Data quality section: %r" % (entry["contributing_reps"], section)
assert str(entry["published_reps"]) in section, "published_reps (%d) not named in Data quality section: %r" % (entry["published_reps"], section)

# Scoped to the Reduced rep sets sub-list specifically -- item_a is
# expected elsewhere in this section (its own quality_tests_pass carries
# an unrelated, pre-existing insufficient_reps entry, same carve-out
# TC-018y/TC-018h use), just never as a REDUCED item.
reduced_start = section.index("Reduced rep sets")
reduced_end = section.index("\n- ", reduced_start + len("Reduced rep sets"))
reduced_section = section[reduced_start:reduced_end]
assert item_a not in reduced_section, "unreduced item %s incorrectly named in the Reduced rep sets list: %r" % (item_a, reduced_section)
' "$out" "$rout" "$item_a" "$item_b" || fail "ff: assertion failed: $(cat "$rout")"

	echo "TC-018ff PASS"
}

# ---------------------------------------------------------------------------
# TC-018gg (UAT round 2, R2-F-3(b), report-baseline.sh side, same fixture
# class as TC-018c: REQ-F-008/REQ-F-009/REQ-N-005): an aggregate with one
# genuinely anomalous record (aggregator exits 1) rendered through report-
# baseline.sh must name the anomalous run_key and outcomes.anomaly_count in
# the Data quality section, and the report itself must exit non-zero --
# matching aggregate-runs.sh's own contract so a pipeline reading only the
# report's exit code cannot publish an anomalous baseline as clean.
# ---------------------------------------------------------------------------
test_gg() {
	local root="$WORKDIR/gg-root"
	local item="f03-fixture-tc018gg"
	place_record "$root" "$item" 1
	place_record "$root" "$item" 2
	place_record "$root" "$item" 3 \
		--unset oracle --unset quality --unset loc \
		--unset sources.oracle --unset sources.quality --unset sources.loc

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -ne 0 ]] || fail "gg: aggregate-runs.sh exited 0, want non-zero (anomaly present)"

	local rout rerr rcode
	{
		read -r rout
		read -r rerr
		read -r rcode
	} < <(run_report "$out")
	[[ "$rcode" -ne 0 ]] || fail "gg: report-baseline.sh exited 0, want non-zero (anomalies[] non-empty must fail the report too)"

	local run_key="${item}::default::rep3"
	local report_text
	report_text="$(cat "$rout")"
	grep -qi "Data quality" <<<"$report_text" || fail "gg: no Data quality section: $report_text"
	grep -q "$run_key" <<<"$report_text" || fail "gg: anomalous run_key $run_key not named in report: $report_text"
	grep -q "anomaly_count" <<<"$report_text" || fail "gg: outcomes.anomaly_count not surfaced in report: $report_text"

	echo "TC-018gg PASS"
}

# ---------------------------------------------------------------------------
# TC-018hh (UAT round 2, R2-F-4, report-baseline.sh side, TC-018cc's own
# fixture: REQ-F-022/AC-21/REQ-N-005): a provenance field legitimately
# absent on EVERY contributing record (model_ids on an all-timeout batch)
# publishes in the aggregate as an explicit `null` plus an
# `unresolved_fields` entry (TC-018cc, T-E40-F03-003's fix). The report must
# still print a "Model IDs" row -- never omit it -- rendered as
# "_not resolvable from the contributing records_" rather than the bare
# Python `None` or a silent drop.
# ---------------------------------------------------------------------------
test_hh() {
	local root="$WORKDIR/hh-root"
	local item_a="f03-fixture-tc018hh-a" item_b="f03-fixture-tc018hh-b"
	place_record "$root" "$item_a" 1 --golden timeout
	place_record "$root" "$item_b" 1 --golden timeout

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -eq 0 ]] || fail "hh: aggregate-runs.sh exited $code, want 0 (model_ids absence explained on every record): $(cat "$err")"

	local rout rerr rcode
	{
		read -r rout
		read -r rerr
		read -r rcode
	} < <(run_report "$out")
	[[ "$rcode" -eq 0 ]] || fail "hh: report-baseline.sh exited $rcode, want 0: $(cat "$rerr")"

	local report_text
	report_text="$(cat "$rout")"
	grep -qi "Model IDs" <<<"$report_text" || fail "hh: no Model IDs row at all -- field silently omitted from the Provenance list: $report_text"
	grep -qi "_not resolvable from the contributing records_" <<<"$report_text" || fail "hh: unresolved provenance field not rendered as explicitly unresolvable: $report_text"
	grep -qi "^- Model IDs: None$" <<<"$report_text" && fail "hh: Model IDs rendered as the bare Python None: $report_text"

	echo "TC-018hh PASS"
}

# ---------------------------------------------------------------------------
# TC-018ii (UAT round 2, R2-F-9, report-baseline.sh side: REQ-F-020/
# REQ-F-021/AC-20): a Class C `cost_usd_total` band whose real min/median/
# max/spread_abs/accept_lo/accept_hi are all sub-1e-6 (min=4e-7, median=
# 5e-7, max=6e-7 -- the UAT report's own repro) must render non-zero
# spread_abs and a non-zero Acceptance interval lower bound under
# significant-digit rendering, rather than the old fixed-6-decimal `fmt()`
# rounding every one of them to a self-contradictory "0". The Class C
# derivation sentence's own trailing "= [lo, hi]." must reproduce the
# printed Acceptance interval exactly, and must not assert a rendered
# "0 (> 0)".
# ---------------------------------------------------------------------------
test_ii() {
	local root="$WORKDIR/ii-root"
	local item="f03-fixture-tc018ii"
	place_record "$root" "$item" 1 --set 'stages[0].usage.total_cost_usd=0' --set 'stages[2].usage.total_cost_usd=4e-07'
	place_record "$root" "$item" 2 --set 'stages[0].usage.total_cost_usd=0' --set 'stages[2].usage.total_cost_usd=5e-07'
	place_record "$root" "$item" 3 --set 'stages[0].usage.total_cost_usd=0' --set 'stages[2].usage.total_cost_usd=6e-07'

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -eq 0 ]] || fail "ii: aggregate-runs.sh exited $code, want 0: $(cat "$err")"

	local rout rerr rcode
	{
		read -r rout
		read -r rerr
		read -r rcode
	} < <(run_report "$out")
	[[ "$rcode" -eq 0 ]] || fail "ii: report-baseline.sh exited $rcode, want 0: $(cat "$rerr")"

	python3 -c '
import json
import re
import sys

aggregate_path, report_path, item = sys.argv[1], sys.argv[2], sys.argv[3]
agg = json.load(open(aggregate_path))
report_text = open(report_path).read()

task = next(t for t in agg["tasks"] if t["item_id"] == item)
block = task["metrics"]["cost_usd_total"]
assert 0 < block["spread_abs"] < 1e-6, block

marker = "#### `cost_usd_total`"
start = report_text.index(marker)
text = report_text[start:]
nxt = re.search(r"\n#### ", text[1:])
if nxt:
    text = text[: nxt.start() + 1]

m = re.search(r"spread_abs=([^,\n]+)", text)
assert m, "no spread_abs on the observed-stats line: %r" % text
printed_spread = m.group(1).strip()
assert printed_spread != "0", "spread_abs rendered as literal 0 though the real value is %r (F-9 regression)" % block["spread_abs"]
assert float(printed_spread) > 0, (printed_spread, block)

im = re.search(r"Acceptance interval: \[([^,]+), ([^\]]+)\]", text)
assert im, "no Acceptance interval line: %r" % text
printed_lo, printed_hi = float(im.group(1)), float(im.group(2))
assert printed_lo > 0, "printed Acceptance interval lower bound rounded to 0 though the published accept_lo is %r" % block["accept_lo"]
assert abs(printed_lo - block["accept_lo"]) < 1e-9, (printed_lo, block["accept_lo"])
assert abs(printed_hi - block["accept_hi"]) < 1e-9, (printed_hi, block["accept_hi"])

dm = re.search(r"- Derivation:.*", text)
assert dm, "no Derivation line: %r" % text
derivation_line = dm.group(0)

rm = re.search(r"r = max - min = (\S+) \(> 0\)", derivation_line)
assert rm, "derivation sentence does not print its own r = ... (> 0) clause: %r" % derivation_line
assert rm.group(1) != "0", "derivation sentence prints r=0 while asserting (> 0) -- self-contradictory: %r" % derivation_line

fm = re.search(r"= \[([^,]+), ([^\]]+)\]\.$", derivation_line)
assert fm, "derivation sentence does not end with its own evaluated [lo, hi]: %r" % derivation_line
rule_lo, rule_hi = float(fm.group(1)), float(fm.group(2))
assert rule_lo == printed_lo and rule_hi == printed_hi, (
    "derivation sentence interval disagrees with the printed Acceptance interval: %r"
    % ((rule_lo, rule_hi, printed_lo, printed_hi),)
)
' "$out" "$rout" "$item" || fail "ii: assertion failed: $(cat "$rout")"

	echo "TC-018ii PASS"
}

# ---------------------------------------------------------------------------
# TC-018 R2-F-10 (round-2 UAT/round-3 code-review finding, never closed by
# the round-2 rework): family_present() tested a family block against
# exactly one unusable value (`null`, TC-018w) and nothing else. `classify()`
# and `inventory[].families_present` must never disagree with the metric
# layer's own `excluded[]` about whether oracle/quality/loc actually
# contributed usable data. Sub-case (a): a non-dict family value (a JSON
# array here) is a structural violation of I-02 -- REQ-F-010, a hard
# fail() like any other malformed record, nothing printed to stdout.
# Sub-case (b): an empty dict is a structurally valid object but carries
# no data -- exactly as absent as a missing/null key -- so an unexplained
# empty oracle reaches the anomaly bucket, is excluded from
# `families_present`, and its own `oracle_f2p_resolved.excluded[]` reason
# agrees (`unexplained_absence`, the anomaly-bucket reason, not
# `family_absent`) -- the R2-F-10 repro's own internal-contradiction
# scenario, closed.
# ---------------------------------------------------------------------------
test_r2f10() {
	# --- (a) non-dict family value ([]) is a hard structural failure. ---
	local root_a="$WORKDIR/r2f10-root-a"
	place_record "$root_a" f03-fixture-r2f10-listval 1 --set 'oracle=[]'

	local out_a err_a code_a
	{
		read -r out_a
		read -r err_a
		read -r code_a
	} < <(run_aggregate "$root_a")
	[[ "$code_a" -ne 0 ]] || fail "r2f10(a): exited 0, want non-zero (oracle: [] is a structural failure, not usable data)"
	[[ ! -s "$out_a" ]] || fail "r2f10(a): stdout not empty on a structural failure: $(cat "$out_a")"
	grep -q "oracle" "$err_a" || fail "r2f10(a): stderr does not name the offending family: $(cat "$err_a")"
	echo "TC-018 R2-F-10(a, non-dict family value is a hard structural failure, nothing printed) PASS"

	# --- (b) empty dict: usable shape, no data -> anomaly, never complete,
	# families_present and excluded[] agree. ---
	local root_b="$WORKDIR/r2f10-root-b"
	place_record "$root_b" f03-fixture-r2f10-emptydict 1 --set 'oracle={}'

	local out_b err_b code_b
	{
		read -r out_b
		read -r err_b
		read -r code_b
	} < <(run_aggregate "$root_b")
	[[ "$code_b" -ne 0 ]] || fail "r2f10(b): exited 0, want non-zero (empty oracle dict is an unexplained absence -> anomaly)"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
run_key = "f03-fixture-r2f10-emptydict::default::rep1"
inv = d["inventory"][run_key]
assert inv["classification"] == "anomaly", inv
assert "oracle" not in inv["families_present"], "empty-dict oracle must not be listed as present: %r" % inv
assert "quality" in inv["families_present"], inv
assert "loc" in inv["families_present"], inv
matches = [a for a in d["anomalies"] if a["run_key"] == run_key]
assert len(matches) == 1 and matches[0]["missing_families"] == ["oracle"], matches
assert d["outcomes"]["anomaly_count"] == 1, d["outcomes"]

task = next(t for t in d["tasks"] if t["item_id"] == "f03-fixture-r2f10-emptydict")
metric = task["metrics"]["oracle_f2p_resolved"]
assert metric["n"] == 0, metric
reasons = {e["reason"] for e in metric["excluded"] if e["run_key"] == run_key}
assert reasons == {"unexplained_absence"}, (
    "families_present/classify() disagree with the metric layer about this record: %r" % metric
)
' "$out_b" || fail "r2f10(b): assertion failed: $(cat "$out_b")"
	echo "TC-018 R2-F-10(b, empty-dict family is anomaly, families_present agrees with excluded[]) PASS"

	echo "TC-018 R2-F-10 PASS"
}

# ---------------------------------------------------------------------------
# TC-018 R2-F-12 (round-2 UAT R2-F-12, code-review round-3 confirmation:
# `is_valid_metric_value()`/`compute_stats()` never closed the round-2
# finding -- a TYPE check where AC-12's invariant requires a DOMAIN check.
# `is_valid_metric_value` accepted any non-bool int/float for Class B/C,
# so a structurally well-typed but out-of-domain value (a negative count,
# a non-finite float) reached `compute_stats` uncaught, and the Class B
# band formula `max(0, min - 1)` silently clamped a negative `min` up to
# 0 -- producing `accept_lo=0 > min=-2` while `excluded == []`, an
# unfalsifiable-looking but actually-violated AC-12 invariant. Sub-case
# (a): the live repro from the finding (reps 0, 0, -2 on a Class B
# metric). Sub-case (b): a non-finite Class C value (NaN), reachable via
# the real JSON-decode path (Python's `json` module accepts bare `NaN`/
# `Infinity`/`-Infinity` tokens by default, so a producer emitting one is
# a real, not hypothetical, input). Sub-case (c): the fix guidance's
# "unfalsifiable rather than fixture-checked" instruction taken literally
# -- `compute_stats()`'s own runtime assertion is invoked DIRECTLY (the
# is_valid_metric_value domain gate bypassed on purpose, simulating a
# future bug that reintroduces an out-of-domain value) and must raise,
# proving the invariant is enforced in compute_stats itself, not merely
# an emergent property of every caller currently filtering correctly.
# ---------------------------------------------------------------------------
extract_compute_stats_module() {
	# The embedded python heredoc's import/constant/function-definition
	# prologue -- everything between the `<<'PYEOF'` marker and the
	# `root, variant_filter, reps_arg_raw = sys.argv[...]` line that starts
	# actually consuming argv -- located by pattern, not a hardcoded line
	# range, so this stays correct as the surrounding script grows. It is
	# side-effect-free at module scope (defs and constants only), so it
	# can be sourced standalone to unit-test `compute_stats` directly,
	# bypassing every caller-side guard on purpose (sub-case c).
	awk '
		/^python3 .*<<.PYEOF./ { capture = 1; next }
		/^root, variant_filter, reps_arg_raw, items_arg_raw = sys\.argv/ { capture = 0 }
		capture { print }
	' "$AGGREGATE" >"$1"
}

test_r2f12() {
	# --- (a) Class B: reps 0, 0, -2 -- the finding's own live repro. ---
	local root_a="$WORKDIR/r2f12-root-a"
	local item_a="f03-fixture-r2f12a"
	place_record "$root_a" "$item_a" 1 --set 'oracle.p2p_regressions_count=0'
	place_record "$root_a" "$item_a" 2 --set 'oracle.p2p_regressions_count=0'
	place_record "$root_a" "$item_a" 3 --set 'oracle.p2p_regressions_count=-2'

	local out_a err_a code_a
	{
		read -r out_a
		read -r err_a
		read -r code_a
	} < <(run_aggregate "$root_a")
	[[ "$code_a" -eq 0 ]] || fail "r2f12(a): exited $code_a, want 0 (an out-of-domain value excludes, never crashes or aborts the whole aggregation): $(cat "$err_a")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item = sys.argv[2]
task = next(t for t in d["tasks"] if t["item_id"] == item)
block = task["metrics"]["p2p_regressions_count"]

# The negative rep must be excluded with a DOMAIN reason, distinct from
# invalid_value_type (it IS a genuine int -- just out of domain).
reasons = {e["run_key"]: e["reason"] for e in block["excluded"]}
assert reasons.get(item + "::default::rep3") == "out_of_domain", reasons

# Only the two in-domain reps (0, 0) contribute.
assert block["n"] == 2, block
assert block["min"] == 0, block
assert block["max"] == 0, block

# AC-12s invariant, over the CONTRIBUTING values only -- the defect this
# case reproduces is exactly accept_lo(0) > min(-2) when the -2 was never
# excluded; with the fix, min is 0 (the -2 never contributes) and the
# invariant holds trivially, but this assertion is what would have
# caught the original defect had it still been present.
assert block["accept_lo"] <= block["min"], block
assert block["accept_hi"] >= block["max"], block
' "$out_a" "$item_a" || fail "r2f12(a): assertion failed: $(cat "$out_a")"
	echo "TC-018 R2-F-12(a, Class B negative value excluded with out_of_domain, AC-12 invariant holds) PASS"

	# --- (b) Class C: a non-finite value (NaN), reachable via the real
	# JSON-decode path. ---
	local root_b="$WORKDIR/r2f12-root-b"
	local item_b="f03-fixture-r2f12b"
	place_record "$root_b" "$item_b" 1 --set 'loc.prod_added=10'
	place_record "$root_b" "$item_b" 2 --set 'loc.prod_added=12'
	place_record "$root_b" "$item_b" 3 --set 'loc.prod_added=NaN'

	local out_b err_b code_b
	{
		read -r out_b
		read -r err_b
		read -r code_b
	} < <(run_aggregate "$root_b")
	[[ "$code_b" -eq 0 ]] || fail "r2f12(b): exited $code_b, want 0: $(cat "$err_b")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item = sys.argv[2]
task = next(t for t in d["tasks"] if t["item_id"] == item)
block = task["metrics"]["loc_prod_added"]

reasons = {e["run_key"]: e["reason"] for e in block["excluded"]}
assert reasons.get(item + "::default::rep3") == "out_of_domain", reasons
assert block["n"] == 2, block
assert block["min"] == 10, block
assert block["max"] == 12, block
assert block["accept_lo"] <= block["min"], block
assert block["accept_hi"] >= block["max"], block
' "$out_b" "$item_b" || fail "r2f12(b): assertion failed: $(cat "$out_b")"
	echo "TC-018 R2-F-12(b, Class C non-finite value excluded with out_of_domain, JSON-decode-reachable) PASS"

	# --- (c)/(d) compute_stats' own runtime invariant check fires when
	# handed an out-of-domain value directly, bypassing every caller-side
	# guard on purpose -- the invariant is enforced IN compute_stats, not
	# merely an emergent property of its callers currently filtering
	# correctly. (NEW-6, uat-20260809-013000-E40-F03.md, round 3): this
	# used to be a bare `assert`, which CPython strips from the compiled
	# bytecode entirely under PYTHONOPTIMIZE=1 (or `-O`/`-OO`) -- silently
	# turning the invariant into a no-op in that mode. It is now an
	# explicit conditional calling this script's own non-elidable
	# fail()/sys.exit(1) mechanism, so (c) (normal interpreter) and (d)
	# (PYTHONOPTIMIZE=1) must both observe the identical failure mode:
	# SystemExit with code 1, never a silent pass.
	local module_py="$WORKDIR/r2f12-compute-stats-module.py"
	extract_compute_stats_module "$module_py"
	[[ -s "$module_py" ]] || fail "r2f12(c): failed to extract compute_stats module from $AGGREGATE"

	local probe='
import sys
sys.path.insert(0, sys.argv[1])
import importlib.util
spec = importlib.util.spec_from_file_location("aggregate_module", sys.argv[2])
mod = importlib.util.module_from_spec(spec)
spec.loader.exec_module(mod)

try:
    # The exact live repro (0, 0, -2), handed to compute_stats() directly
    # -- as if is_valid_metric_value() had NOT excluded the -2 (simulating
    # a future regression). Must call fail()/exit non-zero, never
    # silently publish a band that violates accept_lo <= min -- in EITHER
    # optimization mode.
    mod.compute_stats("B", [0, 0, -2])
except SystemExit as e:
    print("FAIL_RAISED code=%r" % (e.code,))
    sys.exit(0 if e.code == 1 else 1)
else:
    print("NO_FAIL_RAISED -- invariant did not fire")
    sys.exit(1)
'

	local normal_out normal_code
	set +e
	normal_out="$(python3 -c "$probe" "$WORKDIR" "$module_py" 2>&1)"
	normal_code=$?
	set -e
	[[ "$normal_code" -eq 0 ]] || fail "r2f12(c): compute_stats(\"B\", [0, 0, -2]) did not call fail()/sys.exit(1) -- the invariant is not enforced in compute_stats itself: $normal_out"
	[[ "$normal_out" == *FAIL_RAISED* ]] || fail "r2f12(c): unexpected output: $normal_out"
	echo "TC-018 R2-F-12(c, compute_stats' own runtime invariant check fires on a direct out-of-domain call) PASS"

	local optimize_out optimize_code
	set +e
	optimize_out="$(PYTHONOPTIMIZE=1 python3 -c "$probe" "$WORKDIR" "$module_py" 2>&1)"
	optimize_code=$?
	set -e
	[[ "$optimize_code" -eq 0 ]] || fail "r2f12(d): under PYTHONOPTIMIZE=1, compute_stats(\"B\", [0, 0, -2]) did not call fail()/sys.exit(1) -- a bare 'assert' would be silently elided here: $optimize_out"
	[[ "$optimize_out" == *FAIL_RAISED* ]] || fail "r2f12(d): unexpected output: $optimize_out"
	[[ "$normal_code" -eq "$optimize_code" ]] || fail "r2f12(d): normal-mode and PYTHONOPTIMIZE=1 exit codes diverge: $normal_code vs $optimize_code"
	echo "TC-018 R2-F-12(d, invariant check is NOT elided under PYTHONOPTIMIZE=1 -- same failure mode as normal mode) PASS"

	# --- (e) full-pipeline "identical output/exit status" check over a
	# real fixture root under PYTHONOPTIMIZE=1 -- the fix guidance's own
	# literal ask: PYTHONOPTIMIZE=1 must never change aggregate-runs.sh
	# behavior for any real invocation, not just the direct compute_stats
	# probe above. ---
	local root_e="$WORKDIR/r2f12-root-e"
	local item_e="f03-fixture-r2f12e"
	place_record "$root_e" "$item_e" 1
	place_record "$root_e" "$item_e" 2
	place_record "$root_e" "$item_e" 3

	local out_normal err_normal code_normal
	{
		read -r out_normal
		read -r err_normal
		read -r code_normal
	} < <(run_aggregate "$root_e")
	[[ "$code_normal" -eq 0 ]] || fail "r2f12(e): normal-mode run exited $code_normal, want 0: $(cat "$err_normal")"

	local out_pyopt err_pyopt code_pyopt
	out_pyopt="$(mktemp -p "$WORKDIR")"
	err_pyopt="$(mktemp -p "$WORKDIR")"
	set +e
	PYTHONOPTIMIZE=1 "$AGGREGATE" --root "$root_e" >"$out_pyopt" 2>"$err_pyopt"
	code_pyopt=$?
	set -e
	[[ "$code_pyopt" -eq 0 ]] || fail "r2f12(e): PYTHONOPTIMIZE=1 run exited $code_pyopt, want 0: $(cat "$err_pyopt")"
	[[ "$code_normal" -eq "$code_pyopt" ]] || fail "r2f12(e): exit status diverges -- normal=$code_normal PYTHONOPTIMIZE=1=$code_pyopt"
	diff -u "$out_normal" "$out_pyopt" >/dev/null || fail "r2f12(e): PYTHONOPTIMIZE=1 output differs from normal-mode output: $(diff -u "$out_normal" "$out_pyopt")"
	echo "TC-018 R2-F-12(e, full-pipeline output/exit status identical under PYTHONOPTIMIZE=1) PASS"

	echo "TC-018 R2-F-12 PASS"
}

# ---------------------------------------------------------------------------
# TC-018 NEW-2 (uat-20260809-013000-E40-F03.md, round 3: "a declared-matrix
# bound is enforced on one axis of a two-axis matrix, so a reduction along
# the unbounded axis publishes as complete"). TC-018y bounds reps (one
# axis); this closes the other: `--items` bounds the ITEM axis, mirroring
# `--reps`/expected_rep_set exactly and authoritative in both directions.
# Sub-case (a): a declared 2-item matrix where one item has NO record.jsonl
# at all -- must publish missing_run excluded[] entries for it on every
# applicable metric, name it in flags.missing_items[], and withhold
# baseline_id entirely (stricter than TC-018y's own reduced-rep case, which
# still stamps one). Sub-case (b): an item present on disk but NOT in the
# declared set -- must be named in flags.unexpected_items[], its own real
# data left alone (still contributes normally, still lets baseline_id
# publish -- only a MISSING declared item withholds it). Sub-case (c): CLI
# validation of a malformed --items value (an empty token from a stray
# comma) is a usage error (exit 2), same class as --reps's own format
# check.
# ---------------------------------------------------------------------------
test_new2() {
	# --- (a) declared 2-item matrix, one item entirely absent. ---
	local root_a="$WORKDIR/new2-root-a"
	local item_present="f03-fixture-new2-present" item_missing="f03-fixture-new2-missing"
	place_record "$root_a" "$item_present" 1
	place_record "$root_a" "$item_present" 2

	local out_a err_a code_a
	{
		read -r out_a
		read -r err_a
		read -r code_a
	} < <(run_aggregate "$root_a" --reps 2 --items "$item_present,$item_missing")
	[[ "$code_a" -eq 0 ]] || fail "new2(a): exited $code_a, want 0 (a wholly missing declared item is not itself an anomalous record): $(cat "$err_a")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item_present, item_missing = sys.argv[2], sys.argv[3]

assert "baseline_id" not in d, "a declared item with zero records must withhold baseline_id entirely: %r" % d.get("baseline_id")

missing = d["flags"]["missing_items"]
assert missing == [item_missing], missing
assert item_present not in missing, missing

unexpected = d["flags"]["unexpected_items"]
assert unexpected == [], unexpected

task_missing = next(t for t in d["tasks"] if t["item_id"] == item_missing)
missing_key_1 = item_missing + "::default::rep1"
missing_key_2 = item_missing + "::default::rep2"
saw = 0
for metric_id, block in task_missing["metrics"].items():
    reasons = {e["run_key"]: e["reason"] for e in block["excluded"]}
    assert reasons.get(missing_key_1) == "missing_run", "%s: no missing_run entry for rep1: %r" % (metric_id, block["excluded"])
    assert reasons.get(missing_key_2) == "missing_run", "%s: no missing_run entry for rep2: %r" % (metric_id, block["excluded"])
    assert block["n"] == 0, "%s: n=%r, want 0 (zero real records)" % (metric_id, block["n"])
    saw += 1
assert saw > 5, "too few metrics carried the missing_run entries to be a meaningful check: %d" % saw

task_present = next(t for t in d["tasks"] if t["item_id"] == item_present)
for metric_id, block in task_present["metrics"].items():
    assert not any(e["reason"] == "missing_run" for e in block["excluded"]), "%s: unexpected missing_run entry: %r" % (metric_id, block["excluded"])
' "$out_a" "$item_present" "$item_missing" || fail "new2(a): assertion failed: $(cat "$out_a")"
	echo "TC-018 NEW-2(a, declared item with zero records: missing_run everywhere, flags.missing_items, baseline_id withheld) PASS"

	# --- (b) an item present on disk but not in the declared --items set. ---
	local root_b="$WORKDIR/new2-root-b"
	local item_declared="f03-fixture-new2-declared" item_extra="f03-fixture-new2-unexpected"
	place_record "$root_b" "$item_declared" 1
	place_record "$root_b" "$item_declared" 2
	place_record "$root_b" "$item_extra" 1
	place_record "$root_b" "$item_extra" 2

	local out_b err_b code_b
	{
		read -r out_b
		read -r err_b
		read -r code_b
	} < <(run_aggregate "$root_b" --items "$item_declared")
	[[ "$code_b" -eq 0 ]] || fail "new2(b): exited $code_b, want 0: $(cat "$err_b")"

	python3 -c '
import json
import sys

d = json.load(open(sys.argv[1]))
item_declared, item_extra = sys.argv[2], sys.argv[3]

assert d["flags"]["missing_items"] == [], d["flags"]["missing_items"]
assert d["flags"]["unexpected_items"] == [item_extra], d["flags"]["unexpected_items"]

# An unexpected item is undeclared, not invalid -- its own data still
# contributes normally, and (unlike a missing declared item) does not
# withhold baseline_id.
assert "baseline_id" in d, "an unexpected (not missing) item must not withhold baseline_id: %r" % d
task_extra = next(t for t in d["tasks"] if t["item_id"] == item_extra)
for metric_id, block in task_extra["metrics"].items():
    assert not any(e["reason"] == "missing_run" for e in block["excluded"]), "%s: unexpected missing_run entry on a real, just-undeclared item: %r" % (metric_id, block["excluded"])
' "$out_b" "$item_declared" "$item_extra" || fail "new2(b): assertion failed: $(cat "$out_b")"
	echo "TC-018 NEW-2(b, observed item outside the declared set: flags.unexpected_items, baseline_id still publishes) PASS"

	# --- (c) malformed --items (a stray comma) is a usage error. ---
	local out_c err_c code_c
	out_c="$(mktemp -p "$WORKDIR")"
	err_c="$(mktemp -p "$WORKDIR")"
	set +e
	"$AGGREGATE" --root "$root_b" --items "a,,b" >"$out_c" 2>"$err_c"
	code_c=$?
	set -e
	[[ "$code_c" -eq 2 ]] || fail "new2(c): --items \"a,,b\" exited $code_c, want 2 (usage error): $(cat "$err_c")"
	[[ -s "$out_c" ]] && fail "new2(c): stdout must be empty on a usage error: $(cat "$out_c")"
	echo "TC-018 NEW-2(c, malformed --items is a usage error) PASS"

	echo "TC-018 NEW-2 PASS"
}

# ---------------------------------------------------------------------------
# TC-018 NEW-3 (uat-20260809-013000-E40-F03.md, round 3: "enforcement is
# changed and the operator contract that enumerates it is updated on one
# surface only, so a set the document declares closed is open in the
# shipped code"). bench/README.md's "Metric registry and exclusion
# reasons" table (### heading) claims to be the CLOSED set of every
# `excluded[].reason` value aggregate-runs.sh can emit. Rather than
# re-asserting the fix guidance's three literally-named gaps by hand (which
# would silently go stale again the next time a reason is added or
# renamed), this mechanically diffs the table's documented codes against
# the reason string literals aggregate-runs.sh's embedded python actually
# returns as an `excluded[]` reason -- so a FUTURE new reason, or a
# rename/removal of an existing one, fails this test on either side of the
# diff without anyone having to remember to update it by hand.
#
# Extraction is intentionally narrow rather than a general python parse:
#   - `("excluded", "<reason>")` two-tuple literals (the direct return
#     shape most call sites use);
#   - every bare `return "<reason>"` inside `explain_kind()` (its three
#     literals reach excluded[] via `("excluded", ek)` at :773, so `ek`
#     itself is not visible to the tuple-literal regex above); and
#   - every bare `return "<reason>"` inside `metric_value_exclusion_reason()`
#     (its two literals reach excluded[] via
#     `("excluded", metric_value_exclusion_reason(...))` in evaluate_metric,
#     same indirection).
# `unpinned_field` is deliberately excluded from this set: it is a
# `provenance.divergences[]` reason (R2-F-11), not an `excluded[].reason`
# -- a different vocabulary the exclusion-reason table never claims to
# enumerate. It, `unresolved_fields[]`'s own `all-exempt`/`never-present`
# reasons, and replay-manifest.sh's `aggregator_anomaly` are checked
# separately below by simple presence (not a table diff, since README
# documents them in prose, not a closed table).
# ---------------------------------------------------------------------------
test_new3() {
	local diff_out
	diff_out="$(python3 - "$AGGREGATE" "$README_PATH" <<'PYEOF'
import re
import sys

script_path, readme_path = sys.argv[1], sys.argv[2]
script_text = open(script_path).read()
readme_text = open(readme_path).read()


def func_body(name):
    m = re.search(r"^def %s\(.*?\n(.*?)\n\n\n" % re.escape(name), script_text, re.S | re.M)
    if not m:
        raise SystemExit("test_new3: could not locate function %r in %s -- extraction regex is stale" % (name, script_path))
    return m.group(1)


tuple_reasons = set(re.findall(r'\("excluded",\s*"([a-z_]+)"\)', script_text))
explain_kind_reasons = set(re.findall(r'return "([a-z_]+)"', func_body("explain_kind")))
metric_exclusion_reasons = set(re.findall(r'return "([a-z_]+)"', func_body("metric_value_exclusion_reason")))
dict_reasons = set(re.findall(r'"reason":\s*"([a-z_]+)"', script_text)) - {"unpinned_field", "all-exempt", "never-present"}

script_reasons = tuple_reasons | explain_kind_reasons | metric_exclusion_reasons | dict_reasons
if not script_reasons:
    raise SystemExit("test_new3: extracted zero reasons from %s -- extraction regexes are stale" % script_path)

start = readme_text.index("### Metric registry and exclusion reasons")
end = readme_text.index("\n###", start + 10)
section = readme_text[start:end]
readme_reasons = set(re.findall(r"^\| `([a-z_]+)` \|", section, re.M))
if not readme_reasons:
    raise SystemExit("test_new3: extracted zero reasons from README table -- extraction regex or heading text is stale")

missing_from_readme = sorted(script_reasons - readme_reasons)
extra_in_readme = sorted(readme_reasons - script_reasons)

if missing_from_readme or extra_in_readme:
    print("MISMATCH")
    print("emitted-but-undocumented: %s" % missing_from_readme)
    print("documented-but-unemitted: %s" % extra_in_readme)
else:
    print("MATCH")
PYEOF
	)"

	grep -q "^MATCH$" <<<"$diff_out" || fail "new3: README's exclusion-reason table diverges from aggregate-runs.sh's emitted reasons: $diff_out"

	# Prose-documented (not table-enumerated) vocabulary the round-4 sweep
	# also found undocumented: R2-F-4's unresolved_fields[]/its two reason
	# values, R2-F-11's unpinned_field divergence, and replay-manifest.sh's
	# aggregator_anomaly. Simple presence checks, not a mechanical diff --
	# these are prose call-outs, not a closed table.
	grep -q "unresolved_fields\[\]" "$README_PATH" || fail "new3: README does not document provenance.unresolved_fields[]"
	grep -q "all-exempt" "$README_PATH" || fail "new3: README does not document unresolved_fields[]'s all-exempt reason"
	grep -q "never-present" "$README_PATH" || fail "new3: README does not document unresolved_fields[]'s never-present reason"
	grep -q "unpinned_field" "$README_PATH" || fail "new3: README does not document the unpinned_field divergence reason"
	grep -q "aggregator_anomaly" "$README_PATH" || fail "new3: README does not document replay-manifest.sh's aggregator_anomaly reason"

	echo "TC-018 NEW-3 PASS"
}

# ---------------------------------------------------------------------------
# TC-018 NEW-4 (uat-20260809-013000-E40-F03.md, round 3: "a renderer
# prints a positive absence claim ('none.') for a data-quality section
# whose input was never computed at all"). aggregate-runs.sh omits
# `flags{}` ENTIRELY when provenance is non-uniform (see the `if uniform:`
# guard around `aggregate["flags"] = {...}`) -- "not computed" and
# "computed, empty" must never render identically. Reuses TC-018ee's own
# non-uniform fixture shape (a model_ids divergence is enough to make the
# batch non-uniform and omit flags{}); every Data-quality category must
# read "_not computed (provenance is not uniform...)_", never "none.", and
# the per-task noise-band section's "No per-task noise band published"
# sentence must likewise say WHY (suppressed, not "zero tasks").
# ---------------------------------------------------------------------------
test_new4() {
	local root="$WORKDIR/new4-root"
	local item="f03-fixture-tc018new4"
	place_record "$root" "$item" 1
	place_record "$root" "$item" 2 --set 'manifest.model_ids=["claude-evil-9"]'

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -ne 0 ]] || fail "new4: aggregate-runs.sh exited 0, want non-zero (non-uniform provenance)"

	python3 -c '
import json
import sys
d = json.load(open(sys.argv[1]))
assert "flags" not in d, "fixture must exercise the flags-entirely-absent shape: %r" % d.get("flags")
' "$out" || fail "new4: fixture setup assertion failed (flags present when it should be absent): $(cat "$out")"

	local rout rerr rcode
	{
		read -r rout
		read -r rerr
		read -r rcode
	} < <(run_report "$out")
	[[ "$rcode" -ne 0 ]] || fail "new4: report-baseline.sh exited 0, want non-zero (non-uniform batch)"

	local report_text
	report_text="$(cat "$rout")"

	local dq_section
	dq_section="$(python3 -c '
import sys
text = sys.argv[1]
start = text.index("## Data quality")
end = text.index("## Noise band per task")
print(text[start:end])
' "$report_text")"

	# "none." may legitimately appear on the Anomalous-records line
	# (anomalies[] is always computed, see below) -- excluded here so this
	# check is scoped to the flags-derived categories only.
	grep -vi "^- Anomalous records:" <<<"$dq_section" | grep -qi "none\.$" && fail "new4: Data-quality section still contains a literal '\''none.'\'' line over a batch whose flags were never computed: $dq_section"
	grep -qi "not computed" <<<"$dq_section" || fail "new4: Data-quality section does not explain that its categories were not computed: $dq_section"
	grep -qi "provenance is not uniform" <<<"$dq_section" || fail "new4: Data-quality 'not computed' line does not name the reason (non-uniform provenance): $dq_section"

	# Anomalous records draws on `anomalies[]`, a top-level field set
	# unconditionally regardless of `uniform` -- it is genuinely computed
	# here (this fixture has no anomalous record), so "none." is the
	# correct, truthful rendering for THIS category; only the four
	# flags-derived categories below are actually not-computed.
	grep -q "^- Anomalous records: none\.\$" <<<"$dq_section" || fail "new4: Anomalous records should still read 'none.' -- anomalies[] is genuinely computed even when flags{} is not: $dq_section"

	for label in "Reduced rep sets" "Unexpected reps" "Insufficient reps" "Unusable metrics"; do
		grep -q "^- ${label}: _not computed" <<<"$dq_section" || fail "new4: '${label}' does not render the not-computed line: $dq_section"
	done

	grep -qi "No per-task noise band published" <<<"$report_text" || fail "new4: missing the no-tasks sentence: $report_text"
	grep -qi "suppressed" <<<"$report_text" || fail "new4: the no-per-task-noise-band sentence does not distinguish suppression (non-uniform) from a genuinely empty uniform batch: $report_text"

	echo "TC-018 NEW-4 PASS"
}

# ---------------------------------------------------------------------------
# TC-018 NEW-5 (uat-20260809-013000-E40-F03.md, round 3: "the Divergences
# table renders raw Python reprs into published markdown, disagreeing with
# the PROVENANCE_FIELDS loop 3 lines below which renders the same values
# correctly"). Builds a single non-uniform aggregate carrying BOTH failure
# shapes the finding names: a list-valued divergence (model_ids differs
# per record, reproducing `['claude-evil-9']`) and a None-valued
# divergence (fixture_base_sha unset on every contributing record --
# R2-F-11's "unpinned_field" shape, TC-018x(c)'s own fixture, reproducing
# the bare `None`). After the fix, neither raw repr may appear anywhere in
# the report, and the Divergences table's rendering must match the
# PROVENANCE_FIELDS loop's own wording for the same value shapes (shared
# render_value() helper).
# ---------------------------------------------------------------------------
test_new5() {
	local root="$WORKDIR/new5-root"
	local item="f03-fixture-tc018new5"
	place_record "$root" "$item" 1 --unset manifest.fixture_base_sha
	place_record "$root" "$item" 2 --unset manifest.fixture_base_sha --set 'manifest.model_ids=["claude-evil-9"]'

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -ne 0 ]] || fail "new5: aggregate-runs.sh exited 0, want non-zero (non-uniform provenance)"

	python3 -c '
import json
import sys
d = json.load(open(sys.argv[1]))
prov = d["provenance"]
divs = prov.get("divergences", [])
fields = {dv["field"]: dv for dv in divs}
assert "fixture_base_sha" in fields, "fixture setup: expected an unpinned_field (None-valued) divergence: %r" % divs
assert fields["fixture_base_sha"]["values"][0]["value"] is None, fields["fixture_base_sha"]
assert "model_ids" in fields, "fixture setup: expected a list-valued divergence: %r" % divs
assert any(isinstance(v["value"], list) for v in fields["model_ids"]["values"]), fields["model_ids"]
' "$out" || fail "new5: fixture setup assertion failed: $(cat "$out")"

	local rout rerr rcode
	{
		read -r rout
		read -r rerr
		read -r rcode
	} < <(run_report "$out")
	[[ "$rcode" -ne 0 ]] || fail "new5: report-baseline.sh exited 0, want non-zero (non-uniform batch)"

	local report_text
	report_text="$(cat "$rout")"

	grep -qF "['" <<<"$report_text" && fail "new5: report still contains a raw Python list repr (\"['\"): $report_text"
	grep -qE '(^|[^a-zA-Z_])None([^a-zA-Z_]|$)' <<<"$report_text" && fail "new5: report still contains a bare Python None: $report_text"

	grep -q "claude-evil-9" <<<"$report_text" || fail "new5: diverging list value not rendered at all: $report_text"

	# The Divergences table and the PROVENANCE_FIELDS loop must render the
	# same None-shaped absence identically (shared render_value()) -- both
	# sections must use the SAME wording for fixture_base_sha's None,
	# never the table saying one thing and the Provenance list another.
	local divergences_section
	divergences_section="$(python3 -c '
import sys
text = sys.argv[1]
start = text.index("### Divergences")
table_start = text.index("\n\n", start) + 2
end = text.index("\n\n", table_start)
print(text[start:end])
' "$report_text")"
	grep -qi "not resolvable\|absent" <<<"$divergences_section" || fail "new5: Divergences table does not render the None value with an absence phrase: $divergences_section"

	echo "TC-018 NEW-5 PASS"
}

# ---------------------------------------------------------------------------
# TC-018 CR5-1 (code review round 5): over a non-uniform aggregate (flags{}
# absent), the "## Corpus feedback to E40-F01" section unconditionally
# printed "No tasks in this baseline were flagged non-discriminative." --
# an affirmative false-empty claim about an analysis that never ran, the
# same defect class NEW-4 was filed against (see emit_flags_category()
# above), but in a section structurally outside test_new4's ## Data
# quality / ## Noise band per task slice, which is why it escaped there.
# ---------------------------------------------------------------------------
test_cr5_1() {
	local root="$WORKDIR/cr5-1-root"
	local item="f03-fixture-tc018cr51"
	place_record "$root" "$item" 1
	place_record "$root" "$item" 2 --set 'manifest.model_ids=["claude-evil-9"]'

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root")
	[[ "$code" -ne 0 ]] || fail "cr5-1: aggregate-runs.sh exited 0, want non-zero (non-uniform provenance)"

	python3 -c '
import json
import sys
d = json.load(open(sys.argv[1]))
assert "flags" not in d, "fixture must exercise the flags-entirely-absent shape: %r" % d.get("flags")
' "$out" || fail "cr5-1: fixture setup assertion failed (flags present when it should be absent): $(cat "$out")"

	local rout rerr rcode
	{
		read -r rout
		read -r rerr
		read -r rcode
	} < <(run_report "$out")
	[[ "$rcode" -ne 0 ]] || fail "cr5-1: report-baseline.sh exited 0, want non-zero (non-uniform batch)"

	local report_text
	report_text="$(cat "$rout")"

	local corpus_section
	corpus_section="$(python3 -c '
import sys
text = sys.argv[1]
start = text.index("## Corpus feedback to E40-F01")
end = text.index("## Measurement caveats")
print(text[start:end])
' "$report_text")"

	grep -qi "No tasks in this baseline were flagged non-discriminative" <<<"$corpus_section" && fail "cr5-1: Corpus-feedback section still affirmatively claims no non-discriminative tasks over a batch whose flags were never computed: $corpus_section"
	grep -qi "not computed" <<<"$corpus_section" || fail "cr5-1: Corpus-feedback section does not explain that non-discriminative-task detection was not computed: $corpus_section"
	grep -qi "provenance is not uniform" <<<"$corpus_section" || fail "cr5-1: Corpus-feedback 'not computed' line does not name the reason (non-uniform provenance): $corpus_section"

	echo "TC-018 CR5-1 PASS"
}

# ---------------------------------------------------------------------------
# TC-018 CR6-1 (code review round 6): report-baseline.sh's Provenance
# section silently omitted the "Input digest"/"Baseline id" rows entirely
# when input_digest/baseline_id are absent from the aggregate, instead of
# naming the absence the way the PROVENANCE_FIELDS loop three lines above
# already does ("_absent from the aggregate_", the R2-F-2/R2-F-4 fix) --
# the third instance of the "affirmative claim (or silence) built on a
# possibly-absent aggregate key" defect class in this file, after NEW-4
# and CR5-1. Reuses TC-018 NEW-2(a)'s own fixture shape (a declared item
# with zero records, --items-bounded), which NEW-2(a) already proves
# withholds baseline_id on a genuinely uniform, exit-0 aggregate -- this
# is exactly the "looks like a normal, complete report" case the finding
# says a reader has no way to see through. NEW-2(a) itself only asserts on
# the aggregate JSON's own baseline_id key; this sub-case is the first to
# render report-baseline.sh over that shape and inspect the Provenance
# section text.
# ---------------------------------------------------------------------------
test_cr6_1() {
	local root="$WORKDIR/cr6-1-root"
	local item_present="f03-fixture-cr61-present" item_missing="f03-fixture-cr61-missing"
	place_record "$root" "$item_present" 1
	place_record "$root" "$item_present" 2

	local out err code
	{
		read -r out
		read -r err
		read -r code
	} < <(run_aggregate "$root" --items "$item_present,$item_missing")
	[[ "$code" -eq 0 ]] || fail "cr6-1: aggregate-runs.sh exited $code, want 0 (a wholly missing declared item is not itself an anomalous record): $(cat "$err")"

	python3 -c '
import json
import sys
d = json.load(open(sys.argv[1]))
assert "baseline_id" not in d, "fixture setup: expected baseline_id withheld (NEW-2(a) shape -- declared item with zero records): %r" % d.get("baseline_id")
assert "input_digest" in d, "fixture setup: expected input_digest present (computed unconditionally): %r" % d
' "$out" || fail "cr6-1: fixture setup assertion failed: $(cat "$out")"

	local rout rerr rcode
	{
		read -r rout
		read -r rerr
		read -r rcode
	} < <(run_report "$out")
	[[ "$rcode" -eq 0 ]] || fail "cr6-1: report-baseline.sh exited $rcode, want 0 (uniform batch): $(cat "$rerr")"

	local report_text
	report_text="$(cat "$rout")"

	local prov_section
	prov_section="$(python3 -c '
import sys
text = sys.argv[1]
start = text.index("## Provenance")
end = text.index("## Data quality")
print(text[start:end])
' "$report_text")"

	grep -qE '^- Baseline id: _absent from the aggregate_$' <<<"$prov_section" || fail "cr6-1: Provenance section does not explicitly name baseline_id's absence (must mirror the PROVENANCE_FIELDS loop's own wording): $prov_section"
	grep -qE '^- Input digest: `' <<<"$prov_section" || fail "cr6-1: Provenance section does not render input_digest's present value: $prov_section"

	echo "TC-018 CR6-1 PASS"
}

test_new3
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
test_w
test_x
test_y
test_z
test_aa
test_bb
test_cc
test_dd
test_ee
test_ff
test_gg
test_hh
test_ii
test_r2f10
test_r2f12
test_new2
test_new4
test_new5
test_cr5_1
test_cr6_1

echo "TC-018 PASS"
