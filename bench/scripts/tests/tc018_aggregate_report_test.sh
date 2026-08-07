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
# T-E40-F03-004 extends this SAME file with sub-cases g-p, r, u, v (per-
# metric statistics, bands, flags, `input_digest`, and `report-baseline.sh`
# content) -- mirrors tc015's own T-E40-F02-001/-002/-007 extension
# precedent (see that file's own header comment). AC-09's excluded[]/band
# assertions (the per-metric registry) are OUT OF SCOPE here -- this file's
# test_d only exercises AC-09's classification/outcomes slice, which is
# everything `aggregate-runs.sh` computes at this task's stage; the
# per-metric excluded[] half is exercised once T-E40-F03-004 lands the
# metrics registry.
#
# Caller-Path Contract (test-plan.md TC-018): real subprocess invocation of
# `bench/scripts/aggregate-runs.sh --root <dir>` against fixture roots
# built entirely from `bench/scripts/testdata/aggregate/gen_fixtures.py`
# (T-E40-F03-002) -- never a hand-authored record (REQ-N-006, ADR-F03-08).
# Nothing in the aggregator is stubbed; this is the real I-02 consumer path
# (I-02: consumes, contract test tests/contracts/e40_i02_artifact_contract_
# test.go#TC-001, referenced not twinned, ADR-F03-09).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

AGGREGATE="$SCRIPTS_DIR/aggregate-runs.sh"
GEN_FIXTURES="$SCRIPTS_DIR/testdata/aggregate/gen_fixtures.py"

fail() {
	echo "TC-018 FAIL: $1" >&2
	exit 1
}

[[ -x "$AGGREGATE" ]] || fail "aggregate-runs.sh missing or not executable: $AGGREGATE"
[[ -f "$GEN_FIXTURES" ]] || fail "gen_fixtures.py missing: $GEN_FIXTURES"
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
# TC-018d (AC-09, this task's slice -- see file header): a timeout record
# mixed with two completed reps of the same item contributes to `outcomes`
# counts and `timeout_rate` only, classifies explained_absence (never
# anomaly), and the batch exits zero (uniform provenance, no anomaly). The
# per-metric excluded[]/band/`wall_clock_ns` n assertions require the
# metrics registry T-E40-F03-004 adds and are NOT exercised here.
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

test_a
test_b
test_c
test_d
test_e
test_f
test_q
test_s
test_t

echo "TC-018 PASS"
