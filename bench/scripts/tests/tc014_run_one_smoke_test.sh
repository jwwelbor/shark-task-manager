#!/usr/bin/env bash
# TC-014 (test-plan.md AC test matrix; T-E40-F02-003/004 task spec Test
# Cases).
#
# T-E40-F02-003 built sub-cases a, b, c, e, g. T-E40-F02-004 adds sub-cases
# d (F2P dispatch-time leak surface, AC-12) and f (measurement-before-
# injection ordering, AC-18), which exercise the pinned post-run pipeline
# that task extends run-one.sh with (ADR-F02-11).
#
# Caller-Path Contract (test-plan.md TC-014): real subprocess invocation of
# `bench/scripts/run-one.sh` against the real corpus (bench/corpus/corpus.yaml)
# and a real bench/fixture-repo checkout, with the `shark` executable
# resolved ahead of the real binary on PATH for the duration of each test
# (a real, executable stub dispatching on subcommand: create epic/feature/
# task/bug and run), mirroring TC-011's go/golangci-lint PATH-stub
# technique. scripts/shark-scratch-env.sh and its real `admin init` run for
# real (fast, no LLM call) -- only the stub `shark`'s own faked subcommands
# are intercepted; everything else (admin install-shark-data) forwards to
# the real binary. run-one.sh's own bash control flow (phase ordering,
# timeout/process-group handling, artifact directory construction,
# meta.json writing) is never stubbed or bypassed.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"

RUN_ONE="$SCRIPTS_DIR/run-one.sh"
STUB_SHARK="$SCRIPTS_DIR/testdata/stubs/shark"
CORPUS_YAML="$BENCH_DIR/corpus/corpus.yaml"
REAL_SHARK="$REPO_ROOT/bin/shark"

fail() {
	echo "TC-014 FAIL: $1" >&2
	exit 1
}

[[ -x "$RUN_ONE" ]] || fail "run-one.sh missing or not executable"
[[ -x "$STUB_SHARK" ]] || fail "stub shark missing or not executable: $STUB_SHARK"
[[ -x "$REAL_SHARK" ]] || fail "real shark binary missing at $REAL_SHARK (run 'make shark' first)"
[[ -f "$CORPUS_YAML" ]] || fail "corpus.yaml missing: $CORPUS_YAML"
command -v setsid >/dev/null 2>&1 || fail "setsid not found on PATH"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"
command -v pgrep >/dev/null 2>&1 || fail "pgrep not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

STUBBIN="$WORKDIR/stubbin"
mkdir -p "$STUBBIN"
cp "$STUB_SHARK" "$STUBBIN/shark"
chmod +x "$STUBBIN/shark"

# assert_phase_order <stderr-file> -- REQ-N-006 observability: run-one.sh
# prints "run-one: phase=<name>" once per phase, in order.
assert_phase_order() {
	local err_file="$1"
	local line_provision line_seed line_invoke line_postrun
	line_provision="$(grep -n '^run-one: phase=provision$' "$err_file" | head -1 | cut -d: -f1)"
	line_seed="$(grep -n '^run-one: phase=seed$' "$err_file" | head -1 | cut -d: -f1)"
	line_invoke="$(grep -n '^run-one: phase=invoke$' "$err_file" | head -1 | cut -d: -f1)"
	line_postrun="$(grep -n '^run-one: phase=postrun$' "$err_file" | head -1 | cut -d: -f1)"
	[[ -n "$line_provision" && -n "$line_seed" && -n "$line_invoke" && -n "$line_postrun" ]] ||
		fail "missing a phase marker in $err_file (provision=$line_provision seed=$line_seed invoke=$line_invoke postrun=$line_postrun)"
	[[ "$line_provision" -lt "$line_seed" && "$line_seed" -lt "$line_invoke" && "$line_invoke" -lt "$line_postrun" ]] ||
		fail "phase markers out of order in $err_file: provision=$line_provision seed=$line_seed invoke=$line_invoke postrun=$line_postrun"
}

# ---------------------------------------------------------------------------
# TC-014a (AC-01): single-run command, no human interaction, exactly one
# JSONL record at the deterministic path. Negative: a second invocation into
# the same --out refuses rather than silently overwriting.
# ---------------------------------------------------------------------------
test_a() {
	local out_dir="$WORKDIR/a-out" err="$WORKDIR/a.err"

	PATH="$STUBBIN:$PATH" STUB_SHARK_REAL="$REAL_SHARK" \
		"$RUN_ONE" --item cart-remove-item-last-match --variant default --rep 1 \
		--timeout 60 --out "$out_dir" --corpus "$CORPUS_YAML" --skip-canary \
		</dev/null >"$WORKDIR/a.out" 2>"$err" ||
		fail "a: run-one.sh exited non-zero: $(cat "$err")"

	local record="$out_dir/cart-remove-item-last-match/default/rep-1/record.jsonl"
	[[ -f "$record" ]] || fail "a: record.jsonl not found at $record"
	local lines
	lines="$(wc -l <"$record" | tr -d ' ')"
	[[ "$lines" == "1" ]] || fail "a: record.jsonl has $lines lines, want exactly 1"

	assert_phase_order "$err"

	python3 -c '
import json, sys
with open(sys.argv[1]) as f:
    record = json.loads(f.readline())
if record.get("outcome") != "completed":
    sys.exit("TC-014a FAIL: outcome=%r, want completed" % record.get("outcome"))
' "$record" || fail "a: record outcome check failed"

	# G7 reproducibility (architecture.md#metric-collection-and-artifact-schema,
	# E40-interaction-map.md's I-02 row, uat-plan.md UAT-07): meta.json must
	# carry fixture_base_sha/corpus_schema_version/p2p_set/
	# variant_bundle_sha256/shark_version/shark_binary_sha256 -- pinning
	# which fixture SHA, corpus schema, P2P set, workflow bundle content, and
	# shark binary produced this run, so a later replay needs no state
	# outside the artifact directory (ADR-002). Asserted against meta.json
	# directly, not record.jsonl: collect-run.sh's own manifest pass-through
	# is T-E40-F02-001's separate kickback, sequenced after this one.
	local meta="$out_dir/cart-remove-item-last-match/default/rep-1/meta.json"
	[[ -f "$meta" ]] || fail "a: meta.json not found at $meta"
	python3 -c '
import json, re, sys
with open(sys.argv[1]) as f:
    meta = json.load(f)

want_base_sha = "4c24986844b09122e2d516f9bc1ec470b155b441"
if meta.get("fixture_base_sha") != want_base_sha:
    sys.exit("TC-014a FAIL: meta.fixture_base_sha=%r, want %r (corpus.yaml fixture.base_sha)" % (meta.get("fixture_base_sha"), want_base_sha))
if meta.get("corpus_schema_version") != "1.0":
    sys.exit("TC-014a FAIL: meta.corpus_schema_version=%r, want \"1.0\" (corpus.yaml top-level schema_version)" % meta.get("corpus_schema_version"))
if meta.get("p2p_set") != "default":
    sys.exit("TC-014a FAIL: meta.p2p_set=%r, want \"default\" (this items corpus.yaml p2p_set)" % meta.get("p2p_set"))

sha256_re = re.compile(r"^[0-9a-f]{64}$")
bundle_sha = meta.get("variant_bundle_sha256")
if not bundle_sha or not sha256_re.match(bundle_sha):
    sys.exit("TC-014a FAIL: meta.variant_bundle_sha256=%r, want a 64-hex-char sha256 (content hash over the installed workflow bundle, sorted by path)" % bundle_sha)

if not (meta.get("shark_version") or "").strip():
    sys.exit("TC-014a FAIL: meta.shark_version is empty or missing")

bin_sha = meta.get("shark_binary_sha256")
if not bin_sha or not sha256_re.match(bin_sha):
    sys.exit("TC-014a FAIL: meta.shark_binary_sha256=%r, want a 64-hex-char sha256 (sha256sum of the resolved SHARK_BIN)" % bin_sha)
' "$meta" || fail "a: G7 manifest fields check failed"

	# Negative (REQ-F-018): the same --out a second time refuses rather than
	# silently overwriting.
	if PATH="$STUBBIN:$PATH" STUB_SHARK_REAL="$REAL_SHARK" \
		"$RUN_ONE" --item cart-remove-item-last-match --variant default --rep 1 \
		--timeout 60 --out "$out_dir" --corpus "$CORPUS_YAML" --skip-canary \
		</dev/null >"$WORKDIR/a2.out" 2>"$WORKDIR/a2.err"; then
		fail "a: second invocation into the same --out unexpectedly succeeded"
	fi
	grep -qF "$record" "$WORKDIR/a2.err" || fail "a: second invocation's error output doesn't name the existing record path"

	echo "TC-014a PASS"
}

# ---------------------------------------------------------------------------
# TC-014b (AC-11): live-repo isolation. Every stubbed shark invocation's own
# cwd resolves inside the mktemp scratch dir, never the live repo tree; the
# live repo's .sharkconfig.json and git status are byte-identical before and
# after. Static review (mirrors F01's REQ-NF-003 method, recorded not
# re-derived): run-one.sh's only project-initialisation call is
# scripts/shark-scratch-env.sh -- it never invokes `shark admin init` or
# `shark cloud init` directly, and every scratch path it operates against is
# the mktemp directory that script prints. Confirmed by reading run-one.sh.
# ---------------------------------------------------------------------------
test_b() {
	local out_dir="$WORKDIR/b-out" log="$WORKDIR/b-shark.log"
	rm -f "$log"

	local config_before status_before
	config_before="$(sha256sum "$REPO_ROOT/.sharkconfig.json" | awk '{print $1}')"
	status_before="$(cd "$REPO_ROOT" && git status --porcelain)"

	PATH="$STUBBIN:$PATH" STUB_SHARK_REAL="$REAL_SHARK" STUB_SHARK_LOG="$log" \
		"$RUN_ONE" --item cart-remove-item-last-match --variant default --rep 1 \
		--timeout 60 --out "$out_dir" --corpus "$CORPUS_YAML" --skip-canary \
		</dev/null >"$WORKDIR/b.out" 2>"$WORKDIR/b.err" ||
		fail "b: run-one.sh exited non-zero: $(cat "$WORKDIR/b.err")"

	local config_after status_after
	config_after="$(sha256sum "$REPO_ROOT/.sharkconfig.json" | awk '{print $1}')"
	status_after="$(cd "$REPO_ROOT" && git status --porcelain)"

	[[ "$config_before" == "$config_after" ]] || fail "b: live repo .sharkconfig.json changed"
	[[ "$status_before" == "$status_after" ]] || fail "b: live repo git status changed"

	[[ -s "$log" ]] || fail "b: stub shark log is empty -- no invocations recorded"

	python3 -c '
import json, os, sys
log_path, repo_root = sys.argv[1], sys.argv[2]
repo_root = os.path.realpath(repo_root)
count = 0
with open(log_path) as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        entry = json.loads(line)
        cwd = os.path.realpath(entry["cwd"])
        if cwd == repo_root or cwd.startswith(repo_root + os.sep):
            sys.exit("TC-014b FAIL: stubbed shark invocation ran with cwd inside the live repo: %r (argv=%r)" % (cwd, entry["argv"]))
        count += 1
if count == 0:
    sys.exit("TC-014b FAIL: no invocations logged")
' "$log" "$REPO_ROOT" || fail "b: cwd-containment check failed"

	echo "TC-014b PASS"
}

# ---------------------------------------------------------------------------
# TC-014c (AC-16): seeded keys are captured verbatim from the stub's --json
# response, never invented from --item; no create invocation ever passes an
# explicit key.
# ---------------------------------------------------------------------------
test_c() {
	local out_dir="$WORKDIR/c-out" log="$WORKDIR/c-shark.log"
	rm -f "$log"

	PATH="$STUBBIN:$PATH" STUB_SHARK_REAL="$REAL_SHARK" STUB_SHARK_LOG="$log" \
		STUB_SHARK_EPIC_KEY="E77" STUB_SHARK_FEATURE_KEY="E77-F09" STUB_SHARK_TASK_KEY="E77-F09-042" \
		"$RUN_ONE" --item validate-sku-max-length --variant default --rep 1 \
		--timeout 60 --out "$out_dir" --corpus "$CORPUS_YAML" --skip-canary \
		</dev/null >"$WORKDIR/c.out" 2>"$WORKDIR/c.err" ||
		fail "c: run-one.sh exited non-zero: $(cat "$WORKDIR/c.err")"

	local record="$out_dir/validate-sku-max-length/default/rep-1/record.jsonl"
	[[ -f "$record" ]] || fail "c: record.jsonl not found at $record"

	python3 -c '
import json, sys
with open(sys.argv[1]) as f:
    record = json.loads(f.readline())
seeded = record.get("manifest", {}).get("seeded_keys")
want = {"epic": "E77", "feature": "E77-F09", "task": "E77-F09-042"}
if seeded != want:
    sys.exit("TC-014c FAIL: manifest.seeded_keys=%r, want %r" % (seeded, want))
' "$record" || fail "c: seeded_keys check failed"

	python3 -c '
import json, sys
with open(sys.argv[1]) as f:
    for line in f:
        entry = json.loads(line)
        argv = entry["argv"]
        if argv[:1] == ["create"]:
            for a in argv:
                if a in ("--key", "--id") or a.startswith("--key=") or a.startswith("--id="):
                    sys.exit("TC-014c FAIL: create invocation passed an explicit key/id: %r" % argv)
' "$log" || fail "c: explicit-key check failed"

	echo "TC-014c PASS"
}

# ---------------------------------------------------------------------------
# TC-014d (AC-12): F2P dispatch-time leak surface. The stub `shark run`
# invocation inspects its own --workdir at the moment it's invoked -- before
# any post-run injection could have happened -- and records whether the F2P
# path is present then; the primary check is that dispatch-time marker, not
# the checkout's later state (a leak that gets silently overwritten by the
# time the run ends would be invisible to a check that only looks at the
# end state). --keep-scratch keeps the checkout around so this test can
# separately confirm the F2P file IS present once run-one.sh has completed
# (injected post-run, last in the pinned order, T-E40-F02-004).
# ---------------------------------------------------------------------------
test_d() {
	local out_dir="$WORKDIR/d-out" marker="$WORKDIR/d-f2p-marker"
	rm -f "$marker"

	# inventory-reserve-rejects-negative-quantity's single F2P path, per
	# corpus.yaml -- known ahead of dispatch, not derived from the checkout.
	local f2p_rel="pkg/inventory/reserve_negative_quantity_test.go"

	PATH="$STUBBIN:$PATH" STUB_SHARK_REAL="$REAL_SHARK" \
		STUB_SHARK_RUN_F2P_MARKER_FILE="$marker" STUB_SHARK_RUN_F2P_CHECK_PATH="$f2p_rel" \
		"$RUN_ONE" --item inventory-reserve-rejects-negative-quantity --variant default --rep 1 \
		--timeout 60 --out "$out_dir" --corpus "$CORPUS_YAML" --skip-canary --keep-scratch \
		</dev/null >"$WORKDIR/d.out" 2>"$WORKDIR/d.err" ||
		fail "d: run-one.sh exited non-zero: $(cat "$WORKDIR/d.err")"

	[[ -f "$marker" ]] || fail "d: dispatch-time F2P marker was never written (stub 'run' never invoked?)"
	local marker_content
	marker_content="$(cat "$marker")"
	[[ "$marker_content" == "absent" ]] ||
		fail "d: dispatch-time marker says F2P path was '$marker_content', want 'absent' (leaked before dispatch)"

	local meta="$out_dir/inventory-reserve-rejects-negative-quantity/default/rep-1/meta.json"
	[[ -f "$meta" ]] || fail "d: meta.json not found at $meta"
	local scratch_root
	scratch_root="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["scratch_root"])' "$meta")"
	[[ -n "$scratch_root" && -d "$scratch_root" ]] || fail "d: scratch_root missing or not a directory: $scratch_root"
	local checkout_dir="$scratch_root/checkout"
	[[ -f "$checkout_dir/$f2p_rel" ]] ||
		fail "d: F2P path not present in the checkout after run-one.sh completed (post-run injection didn't happen): $checkout_dir/$f2p_rel"

	rm -rf "$scratch_root"

	echo "TC-014d PASS"
}

# ---------------------------------------------------------------------------
# TC-014e (AC-03, AC-04): the timeout kill path is real -- run-one.sh's own
# process-group signaling (setsid + kill -TERM/-KILL -pgid), not simulated
# by the test harness. A stub `shark run` that ignores SIGTERM and forks a
# grandchild proves the WHOLE group dies, not only the direct child.
# ---------------------------------------------------------------------------
test_e() {
	local out_dir="$WORKDIR/e-out"
	local heartbeat="$WORKDIR/e-heartbeat"
	local pgid_file="$heartbeat.pgid"
	local err="$WORKDIR/e.err"
	rm -f "$heartbeat" "$pgid_file"

	# Backgrounded (not synchronous): the "grandchild ticking" proof below is
	# observed WHILE run-one.sh is still inside its cap+grace window, not
	# inferred after the fact from postrun timing (collect-run.sh's own
	# runtime would otherwise smuggle timing noise into a post-hoc
	# mtime-vs-now comparison).
	RUN_ONE_KILL_GRACE_S=2 \
		PATH="$STUBBIN:$PATH" STUB_SHARK_REAL="$REAL_SHARK" \
		STUB_SHARK_RUN_HANG=1 STUB_SHARK_RUN_FORK_GRANDCHILD=1 STUB_SHARK_HEARTBEAT_FILE="$heartbeat" \
		"$RUN_ONE" --item cart-remove-item-last-match --variant default --rep 1 \
		--timeout 2 --out "$out_dir" --corpus "$CORPUS_YAML" --skip-canary \
		</dev/null >"$WORKDIR/e.out" 2>"$err" &
	local driver_pid=$!

	# Positive (AC-04): poll for the heartbeat to appear while run-one.sh is
	# still running -- proves the grandchild was genuinely alive and
	# ticking before the cap fired, not merely that a file with that name
	# eventually existed by the time everything was already dead.
	local waited_ms=0 ticked="false"
	while [[ "$waited_ms" -lt 10000 ]]; do
		if [[ -f "$heartbeat" && -f "$pgid_file" ]]; then
			ticked="true"
			break
		fi
		sleep 0.1
		waited_ms=$((waited_ms + 100))
	done
	[[ "$ticked" == "true" ]] || fail "e: heartbeat/pgid file never appeared while run-one.sh was still running (grandchild never started ticking)"
	local pgid
	pgid="$(cat "$pgid_file")"
	[[ "$pgid" =~ ^[0-9]+$ ]] || fail "e: captured pgid is not numeric: $pgid"

	wait "$driver_pid" ||
		fail "e: run-one.sh exited non-zero for a timed-out run (a timeout is a recorded outcome, not a driver failure): $(cat "$err")"

	# AC-04 negative: the heartbeat file's mtime must have stopped advancing
	# once the cap+grace fired -- a naive SIGTERM-to-shark-only
	# implementation would leave it still growing. Comparing two samples
	# taken after run-one.sh has already returned (rather than either
	# sample against "now") makes this immune to how long the postrun
	# phase itself took.
	local mtime1 mtime2
	mtime1="$(stat -c %Y "$heartbeat" 2>/dev/null || echo 0)"
	sleep 2
	mtime2="$(stat -c %Y "$heartbeat" 2>/dev/null || echo 0)"
	[[ "$mtime1" == "$mtime2" ]] || fail "e: heartbeat file still advancing after cap+grace fired ($mtime1 -> $mtime2): grandchild not killed"

	# AC-04 primary check: process-group inspection, not run-one.sh's own
	# exit code.
	if pgrep -g "$pgid" >/dev/null 2>&1; then
		fail "e: process group $pgid still has live members after the cap+grace fired (orphan-process defect)"
	fi

	local record="$out_dir/cart-remove-item-last-match/default/rep-1/record.jsonl"
	[[ -f "$record" ]] || fail "e: record.jsonl not written for the timed-out run"
	python3 -c '
import json, sys
with open(sys.argv[1]) as f:
    record = json.loads(f.readline())
if record.get("outcome") != "timeout":
    sys.exit("TC-014e FAIL: outcome=%r, want timeout" % record.get("outcome"))
' "$record" || fail "e: record outcome check failed"

	echo "TC-014e PASS"
}

# ---------------------------------------------------------------------------
# TC-014f (AC-18): pinned measurement-before-injection ordering
# (ADR-F02-11). Positive half: a real run-one.sh run (stub `shark run` =
# completed, zero code changes) against the real fixture checkout --
# build-ledgers.sh, diff-ledgers.sh, and git diff --numstat all run for
# real, never stubbed. Counter-factual half: NOT a run-one.sh invocation
# (its order is fixed by construction) -- a documented test double that
# runs the SAME real tools in the deliberately WRONG order (inject, then
# measure) over an independently checked-out fixture tree, proving the
# positive-half assertion is actually sensitive to ordering rather than
# vacuously true (a stubbed-zero measurement tool would pass either way).
# ---------------------------------------------------------------------------
test_f() {
	local out_dir="$WORKDIR/f-out"

	PATH="$STUBBIN:$PATH" STUB_SHARK_REAL="$REAL_SHARK" \
		"$RUN_ONE" --item cart-remove-item-last-match --variant default --rep 1 \
		--timeout 60 --out "$out_dir" --corpus "$CORPUS_YAML" --skip-canary --keep-scratch \
		</dev/null >"$WORKDIR/f.out" 2>"$WORKDIR/f.err" ||
		fail "f: run-one.sh exited non-zero: $(cat "$WORKDIR/f.err")"

	local run_dir="$out_dir/cart-remove-item-last-match/default/rep-1"
	local record="$run_dir/record.jsonl"
	[[ -f "$record" ]] || fail "f: record.jsonl not found"
	local meta="$run_dir/meta.json"
	[[ -f "$meta" ]] || fail "f: meta.json not found"

	local scratch_root
	scratch_root="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["scratch_root"])' "$meta")"
	[[ -n "$scratch_root" && -d "$scratch_root" ]] || fail "f: scratch_root missing or not a directory: $scratch_root"
	local checkout_dir="$scratch_root/checkout"

	# Presence, not merely absence-implies-zero -- the vacuous-truth hole
	# this TC exists to close: every pinned-order post-run artifact was
	# genuinely produced before checking any of its values equal zero.
	local f
	for f in post/numstat.txt post/quality.json post/toolchain-guard.json \
		post/test-diff.json post/lint-diff.json post/tests.json post/lint.json; do
		[[ -f "$run_dir/$f" ]] || fail "f: expected post-run artifact missing: $run_dir/$f"
	done

	python3 -c '
import json, sys
with open(sys.argv[1]) as f:
    record = json.loads(f.readline())
loc = record.get("loc")
required = ("prod_added", "prod_deleted", "test_added", "test_deleted", "files_touched")
if not loc or not all(k in loc for k in required):
    sys.exit("TC-014f FAIL: record.loc missing or incomplete: %r" % (loc,))
for key in required:
    if loc[key] != 0:
        sys.exit("TC-014f FAIL: loc[%s]=%r, want 0 (the context mirror and F2P injection must both be excluded from the pre-injection measurement)" % (key, loc[key]))
quality = record.get("quality") or {}
if "lint_new_issues_count" not in quality:
    sys.exit("TC-014f FAIL: record.quality missing lint_new_issues_count entirely")
if quality["lint_new_issues_count"] != 0:
    sys.exit("TC-014f FAIL: quality.lint_new_issues_count=%r, want 0 (measured before F2P injection)" % quality["lint_new_issues_count"])
' "$record" || fail "f: positive-case record assertion failed"

	# F2P file IS present in the checkout now (injected last, per the
	# pinned order) -- proves this isn't vacuously true because injection
	# never happened at all.
	local f2p_rel="pkg/cart/remove_item_last_test.go"
	[[ -f "$checkout_dir/$f2p_rel" ]] || fail "f: F2P file not present post-run: $checkout_dir/$f2p_rel"

	rm -rf "$scratch_root"

	# --- counter-factual: the SAME real tools, deliberately wrong order ----
	local cf_dir="$WORKDIR/f-counterfactual"
	"$SCRIPTS_DIR/checkout-fixture.sh" "4c24986844b09122e2d516f9bc1ec470b155b441" "$cf_dir" \
		>"$WORKDIR/f-cf-checkout.err" 2>&1 ||
		fail "f: counter-factual checkout-fixture.sh failed: $(cat "$WORKDIR/f-cf-checkout.err")"

	# A deliberately non-empty, deliberately lint-dirty synthetic file
	# standing in for "an F2P file injected before measurement" -- this
	# TC's own documented counter-factual (test-plan.md), not a normal test
	# path. Real corpus F2P files are themselves lint-clean (screened by
	# the admission gate's own P2P checks), so a synthetic dirty file is
	# what actually proves the lint half of the ordering assertion instead
	# of relying on an incidental property of unrelated corpus content.
	cat >"$cf_dir/pkg/inventory/zzz_counterfactual_test.go" <<'GOEOF'
package inventory

import "testing"

func TestZZZCounterfactualLintMarker(t *testing.T) {
	x := 1
	x = 2
	_ = x
}
GOEOF

	local cf_post="$WORKDIR/f-cf-post"
	mkdir -p "$cf_post"
	(cd "$cf_dir" && git add -A -N && git diff --numstat "4c24986844b09122e2d516f9bc1ec470b155b441") >"$cf_post/numstat.txt"

	local cf_test_added
	cf_test_added="$(python3 -c '
import sys
total = 0
with open(sys.argv[1]) as f:
    for line in f:
        parts = line.rstrip("\n").split("\t")
        if len(parts) == 3 and parts[2].endswith("_test.go") and parts[0] != "-":
            total += int(parts[0])
print(total)
' "$cf_post/numstat.txt")"
	[[ "$cf_test_added" -gt 0 ]] ||
		fail "f: counter-factual test_added=$cf_test_added, want > 0 (the LOC assertion above would not have caught a wrong-order implementation)"

	"$SCRIPTS_DIR/build-ledgers.sh" "$cf_dir" "$cf_post" >"$WORKDIR/f-cf-build.err" 2>&1 ||
		fail "f: counter-factual build-ledgers.sh failed: $(cat "$WORKDIR/f-cf-build.err")"
	local base_lint="$BENCH_DIR/corpus/ledgers/4c24986844b09122e2d516f9bc1ec470b155b441/lint.json"
	local cf_lint_diff
	cf_lint_diff="$("$SCRIPTS_DIR/diff-ledgers.sh" --kind=lint --base="$base_lint" --post="$cf_post/lint.json")" ||
		fail "f: counter-factual diff-ledgers.sh --kind=lint failed"
	local cf_new_issues_count
	cf_new_issues_count="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["new_issues_count"])' "$cf_lint_diff")"
	[[ "$cf_new_issues_count" -gt 0 ]] ||
		fail "f: counter-factual lint_new_issues_count=$cf_new_issues_count, want > 0 (the lint assertion above would not have caught a wrong-order implementation)"

	rm -rf "$cf_dir"

	echo "TC-014f PASS"
}

# ---------------------------------------------------------------------------
# TC-014g (AC-21): canary invoked by default, aborting before provisioning
# on failure; --skip-canary suppresses the invocation entirely; meta.json
# always records skip_canary explicitly (never merely omits the key).
# ---------------------------------------------------------------------------
test_g() {
	local canary_stub="$STUBBIN/canary-runsurface.sh"

	# (i) explicit override, canary exits 0 -> provisioning proceeds normally.
	cp "$SCRIPTS_DIR/testdata/stubs/canary-pass.sh" "$canary_stub"
	chmod +x "$canary_stub"

	local out1="$WORKDIR/g1-out" inv1="$WORKDIR/g1-invocations"
	: >"$inv1"
	PATH="$STUBBIN:$PATH" STUB_SHARK_REAL="$REAL_SHARK" STUB_CANARY_INVOCATIONS="$inv1" CANARY_BIN="$canary_stub" \
		"$RUN_ONE" --item cart-remove-item-last-match --variant default --rep 1 \
		--timeout 60 --out "$out1" --corpus "$CORPUS_YAML" \
		</dev/null >"$WORKDIR/g1.out" 2>"$WORKDIR/g1.err" ||
		fail "g(i): run-one.sh exited non-zero with a passing canary: $(cat "$WORKDIR/g1.err")"
	[[ -s "$inv1" ]] || fail "g(i): canary was never invoked"
	local meta1="$out1/cart-remove-item-last-match/default/rep-1/meta.json"
	[[ -f "$meta1" ]] || fail "g(i): meta.json not written"
	python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d.get("skip_canary") is False, d' "$meta1" ||
		fail "g(i): meta.json skip_canary is not explicitly false"

	# (ii) explicit override, canary exits 1 naming a field -> aborts BEFORE
	# provisioning.
	cp "$SCRIPTS_DIR/testdata/stubs/canary-fail.sh" "$canary_stub"
	chmod +x "$canary_stub"

	local out2="$WORKDIR/g2-out" inv2="$WORKDIR/g2-invocations"
	: >"$inv2"
	if PATH="$STUBBIN:$PATH" STUB_SHARK_REAL="$REAL_SHARK" STUB_CANARY_INVOCATIONS="$inv2" CANARY_BIN="$canary_stub" \
		"$RUN_ONE" --item cart-remove-item-last-match --variant default --rep 1 \
		--timeout 60 --out "$out2" --corpus "$CORPUS_YAML" \
		</dev/null >"$WORKDIR/g2.out" 2>"$WORKDIR/g2.err"; then
		fail "g(ii): run-one.sh unexpectedly exited 0 with a failing canary"
	fi
	grep -q "stages_completed" "$WORKDIR/g2.err" || fail "g(ii): canary's named field did not appear in run-one.sh's own error output"
	[[ ! -d "$out2" ]] || fail "g(ii): --out directory was created despite the canary failing"
	! grep -q '^run-one: phase=provision$' "$WORKDIR/g2.err" || fail "g(ii): provisioning phase marker appeared despite the canary failing"

	# (iii) --skip-canary -> canary never invoked, even though it would fail.
	local out3="$WORKDIR/g3-out" inv3="$WORKDIR/g3-invocations"
	: >"$inv3"
	PATH="$STUBBIN:$PATH" STUB_SHARK_REAL="$REAL_SHARK" STUB_CANARY_INVOCATIONS="$inv3" CANARY_BIN=/usr/bin/false \
		"$RUN_ONE" --item cart-remove-item-last-match --variant default --rep 1 \
		--timeout 60 --out "$out3" --corpus "$CORPUS_YAML" --skip-canary \
		</dev/null >"$WORKDIR/g3.out" 2>"$WORKDIR/g3.err" ||
		fail "g(iii): run-one.sh exited non-zero with --skip-canary: $(cat "$WORKDIR/g3.err")"
	[[ ! -s "$inv3" ]] || fail "g(iii): canary was invoked despite --skip-canary"
	local meta3="$out3/cart-remove-item-last-match/default/rep-1/meta.json"
	[[ -f "$meta3" ]] || fail "g(iii): meta.json not written"
	python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d.get("skip_canary") is True, d' "$meta3" ||
		fail "g(iii): meta.json skip_canary is not explicitly true"

	echo "TC-014g PASS"
}

# TC-014h: real Shark creates the hierarchy while only `shark run` is
# stubbed. The stub checks the generated task document in --workdir at the
# instant of dispatch and also confirms the held-back F2P test is still absent.
test_h() {
	local out_dir="$WORKDIR/h-out" spec_marker="$WORKDIR/h-spec-marker" f2p_marker="$WORKDIR/h-f2p-marker"
	local f2p_rel="pkg/validate/sku_length_test.go"
	rm -f "$spec_marker" "$f2p_marker"

	PATH="$STUBBIN:$PATH" STUB_SHARK_REAL="$REAL_SHARK" STUB_SHARK_CREATE_REAL=1 \
		STUB_SHARK_RUN_TASK_SPEC_MARKER_FILE="$spec_marker" \
		STUB_SHARK_RUN_TASK_SPEC_TEXT=$'validate.SKU currently accepts a SKU of any length. Add a maximum length\nof 40 characters, returning a clear, descriptive error for longer values\nwhile preserving the existing non-empty and no-whitespace checks.' \
		STUB_SHARK_RUN_F2P_MARKER_FILE="$f2p_marker" STUB_SHARK_RUN_F2P_CHECK_PATH="$f2p_rel" \
		"$RUN_ONE" --item validate-sku-max-length --variant default --rep 1 \
		--timeout 60 --out "$out_dir" --corpus "$CORPUS_YAML" --skip-canary \
		</dev/null >"$WORKDIR/h.out" 2>"$WORKDIR/h.err" ||
		fail "h: run-one.sh exited non-zero: $(cat "$WORKDIR/h.err")"

	[[ "$(cat "$spec_marker")" == "present" ]] ||
		fail "h: seeded task document was not visible with its full 40-character requirement at dispatch: $(cat "$spec_marker" 2>/dev/null || true)"
	[[ "$(cat "$f2p_marker")" == "absent" ]] ||
		fail "h: held-back F2P test was present at dispatch: $(cat "$f2p_marker" 2>/dev/null || true)"

	echo "TC-014h PASS"
}

# TC-014i: with no CANARY_BIN override and no scripts directory on PATH,
# run-one reaches its bundled sibling canary. The canary fixture seam avoids a
# second live dispatch here; TC-016 owns that real canary invocation.
test_i() {
	local out_dir="$WORKDIR/i-out" canary_fixture="$WORKDIR/i-canary.json"
	cat >"$canary_fixture" <<'JSON'
{"entity_key":"T-E01-F01-001","final_status":"completed","stages_completed":0,"stages":[],"outcome":"completed","total_duration_ns":1}
JSON
	case ":$PATH:" in
	*":$SCRIPTS_DIR:"*) fail "i: inherited PATH contains $SCRIPTS_DIR; cannot prove sibling-canary resolution" ;;
	esac

	PATH="$STUBBIN:${PATH#*:}" STUB_SHARK_REAL="$REAL_SHARK" \
		CANARY_RUNSURFACE_RUNRESULT_FIXTURE="$canary_fixture" \
		"$RUN_ONE" --item cart-remove-item-last-match --variant default --rep 1 \
		--timeout 60 --out "$out_dir" --corpus "$CORPUS_YAML" \
		</dev/null >"$WORKDIR/i.out" 2>"$WORKDIR/i.err" ||
		fail "i: run-one.sh did not reach the bundled sibling canary without a CANARY_BIN override: $(cat "$WORKDIR/i.err")"
	grep -q 'running X-07 canary preflight (' "$WORKDIR/i.err" ||
		fail "i: default canary preflight marker missing"

	echo "TC-014i PASS"
}

# TC-014j: a crashed attempt can leave artifacts without record.jsonl. A rerun
# must refuse that non-empty deterministic directory rather than mixing stale
# stdout/post data into a new record.
test_j() {
	local out_dir="$WORKDIR/j-out"
	local run_dir="$out_dir/cart-remove-item-last-match/default/rep-1"
	mkdir -p "$run_dir/run"
	printf 'stale artifact\n' >"$run_dir/run/stderr.ndjson"
	if PATH="$STUBBIN:$PATH" STUB_SHARK_REAL="$REAL_SHARK" \
		"$RUN_ONE" --item cart-remove-item-last-match --variant default --rep 1 \
		--timeout 60 --out "$out_dir" --corpus "$CORPUS_YAML" --skip-canary \
		</dev/null >"$WORKDIR/j.out" 2>"$WORKDIR/j.err"; then
		fail "j: rerun into a stale non-empty run directory unexpectedly succeeded"
	fi
	grep -qF "$run_dir" "$WORKDIR/j.err" || fail "j: refusal does not name stale run directory"
	[[ ! -f "$run_dir/record.jsonl" ]] || fail "j: rerun wrote a record into stale directory"

	local hidden_out="$WORKDIR/j-hidden-out"
	local hidden_run="$hidden_out/cart-remove-item-last-match/default/rep-1"
	mkdir -p "$hidden_run"
	: >"$hidden_run/.stale"
	if PATH="$STUBBIN:$PATH" STUB_SHARK_REAL="$REAL_SHARK" \
		"$RUN_ONE" --item cart-remove-item-last-match --variant default --rep 1 \
		--timeout 60 --out "$hidden_out" --corpus "$CORPUS_YAML" --skip-canary \
		</dev/null >"$WORKDIR/j-hidden.out" 2>"$WORKDIR/j-hidden.err"; then
		fail "j: rerun into a hidden-only stale run directory unexpectedly succeeded"
	fi

	local empty_out="$WORKDIR/j-empty-out"
	local empty_run="$empty_out/cart-remove-item-last-match/default/rep-1"
	mkdir -p "$empty_run"
	PATH="$STUBBIN:$PATH" STUB_SHARK_REAL="$REAL_SHARK" \
		"$RUN_ONE" --item cart-remove-item-last-match --variant default --rep 1 \
		--timeout 60 --out "$empty_out" --corpus "$CORPUS_YAML" --skip-canary \
		</dev/null >"$WORKDIR/j-empty.out" 2>"$WORKDIR/j-empty.err" ||
		fail "j: empty pre-created run directory should remain usable: $(cat "$WORKDIR/j-empty.err")"
	[[ -f "$empty_run/record.jsonl" ]] || fail "j: empty pre-created run directory did not receive a record"
	echo "TC-014j PASS"
}

# TC-014k: a huge failing ledger diagnostic must not overflow exec argv and
# prevent the postrun-abort marker/record from being written.
test_k() {
	local out_dir="$WORKDIR/k-out" ledger_stub="$STUBBIN/build-ledgers-fail.sh"
	cat >"$ledger_stub" <<'EOF'
#!/usr/bin/env bash
head -c 3000000 /dev/zero | tr '\0' x >&2
exit 1
EOF
	chmod +x "$ledger_stub"
	PATH="$STUBBIN:$PATH" STUB_SHARK_REAL="$REAL_SHARK" BUILD_LEDGERS_BIN="$ledger_stub" \
		"$RUN_ONE" --item cart-remove-item-last-match --variant default --rep 1 \
		--timeout 60 --out "$out_dir" --corpus "$CORPUS_YAML" --skip-canary --keep-scratch \
		</dev/null >"$WORKDIR/k.out" 2>"$WORKDIR/k.err" ||
		fail "k: post-run ledger failure must still produce a record: $(cat "$WORKDIR/k.err")"
	local run_dir="$out_dir/cart-remove-item-last-match/default/rep-1"
	[[ -f "$run_dir/post/postrun-abort.json" && -f "$run_dir/record.jsonl" ]] ||
		fail "k: post-run abort marker or record missing after huge diagnostic"
	python3 - "$run_dir/record.jsonl" <<'PYEOF'
import json
import sys
record = json.load(open(sys.argv[1]))
if record.get("quality", {}).get("postrun_abort") != "build_ledgers":
    sys.exit("TC-014k FAIL: missing build_ledgers abort: %r" % record)
if not any(e.get("kind") == "postrun_check_aborted" for e in record.get("errors", [])):
    sys.exit("TC-014k FAIL: abort error absent: %r" % record.get("errors"))
PYEOF
	local scratch_root
	scratch_root="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["scratch_root"])' "$run_dir/meta.json")"
	[[ ! -f "$run_dir/post/f2p.json" ]] || fail "k: F2P/oracle ran after an aborted ledger phase"
	[[ ! -f "$scratch_root/checkout/pkg/cart/remove_item_last_test.go" ]] || fail "k: F2P file injected after aborted ledger phase"
	rm -rf "$scratch_root"
	echo "TC-014k PASS"
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

echo "TC-014: all sub-cases PASS"
