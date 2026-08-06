#!/usr/bin/env bash
# TC-014 (test-plan.md AC test matrix; T-E40-F02-003 task spec Test Cases).
#
# T-E40-F02-003's slice: sub-cases a, b, c, e, g. Sub-cases d (F2P
# dispatch-time leak surface) and f (measurement-before-injection ordering)
# belong to T-E40-F02-004, which extends run-one.sh with the pinned
# post-run pipeline those sub-cases exercise.
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
# TC-014g (AC-21): canary invoked by default, aborting before provisioning
# on failure; --skip-canary suppresses the invocation entirely; meta.json
# always records skip_canary explicitly (never merely omits the key).
# ---------------------------------------------------------------------------
test_g() {
	local canary_stub="$STUBBIN/canary-runsurface.sh"

	# (i) default flags, canary exits 0 -> provisioning proceeds normally.
	cat >"$canary_stub" <<'EOF'
#!/usr/bin/env bash
echo "invoked" >>"${STUB_CANARY_INVOCATIONS:?}"
exit 0
EOF
	chmod +x "$canary_stub"

	local out1="$WORKDIR/g1-out" inv1="$WORKDIR/g1-invocations"
	: >"$inv1"
	PATH="$STUBBIN:$PATH" STUB_SHARK_REAL="$REAL_SHARK" STUB_CANARY_INVOCATIONS="$inv1" \
		"$RUN_ONE" --item cart-remove-item-last-match --variant default --rep 1 \
		--timeout 60 --out "$out1" --corpus "$CORPUS_YAML" \
		</dev/null >"$WORKDIR/g1.out" 2>"$WORKDIR/g1.err" ||
		fail "g(i): run-one.sh exited non-zero with a passing canary: $(cat "$WORKDIR/g1.err")"
	[[ -s "$inv1" ]] || fail "g(i): canary was never invoked"
	local meta1="$out1/cart-remove-item-last-match/default/rep-1/meta.json"
	[[ -f "$meta1" ]] || fail "g(i): meta.json not written"
	python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d.get("skip_canary") is False, d' "$meta1" ||
		fail "g(i): meta.json skip_canary is not explicitly false"

	# (ii) default flags, canary exits 1 naming a field -> aborts BEFORE
	# provisioning.
	cat >"$canary_stub" <<'EOF'
#!/usr/bin/env bash
echo "invoked" >>"${STUB_CANARY_INVOCATIONS:?}"
echo "canary: RunResult field 'stages_completed' is missing" >&2
exit 1
EOF
	chmod +x "$canary_stub"

	local out2="$WORKDIR/g2-out" inv2="$WORKDIR/g2-invocations"
	: >"$inv2"
	if PATH="$STUBBIN:$PATH" STUB_SHARK_REAL="$REAL_SHARK" STUB_CANARY_INVOCATIONS="$inv2" \
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
	PATH="$STUBBIN:$PATH" STUB_SHARK_REAL="$REAL_SHARK" STUB_CANARY_INVOCATIONS="$inv3" \
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

test_a
test_b
test_c
test_e
test_g

echo "TC-014: all sub-cases PASS"
