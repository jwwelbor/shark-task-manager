#!/usr/bin/env bash
# TC-079 (batch half) / T-E40-F10-004: zero-provider-call operator preview
# for run-lifecycle-batch.sh (spec.md REQ-F-001, REQ-F-014, REQ-NF-002;
# test-plan.md TC-079). The comparison-driver half of TC-079
# (run-review-comparison.sh --mode preview) is added to THIS SAME FILE by
# T-E40-F10-005, which does not yet exist -- see that task's Scope note.
#
# Caller-Path Contract (test-plan.md table): only the enumerated
# PATH-shim provider/network denial process is a stub; the
# `run-lifecycle.sh --mode dry-run` invocation itself is real. Matching
# established precedent (tc061_lifecycle_runner_loop_test.sh), the `shark`
# executable it calls is stubbed on PATH -- `shark` is not a denied
# provider/network binary, and stubbing the CLI binary under test is not the
# same as mocking the preview-content assembly, stage resolution, or the
# dry-run delegation call, none of which are mocked here.
#
# This file also proves this task's own local AC-T2 ("a repeat of an
# already-retained (scenario, rep) is classified and skipped;
# --reclaim-incomplete quarantines rather than deletes") at the bottom,
# beyond TC-079's own preview-only body -- the same "additional coverage
# requested for this task" pattern tc080_spend_gate_refusal_test.sh already
# uses for T-E40-F10-003's AC-T3.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$SCRIPTS_DIR/../.." && pwd)"
BATCH="$SCRIPTS_DIR/run-lifecycle-batch.sh"
NEXT_RESPONSE="$SCRIPTS_DIR/testdata/lifecycle/next-response-complete.json"

fail() {
	echo "TC-079 FAIL: $1" >&2
	exit 1
}

[[ -x "$BATCH" ]] || fail "bench/scripts/run-lifecycle-batch.sh missing or not executable"
[[ -f "$NEXT_RESPONSE" ]] || fail "shared lifecycle test fixture missing: $NEXT_RESPONSE"

# shellcheck source=lib/path-shim-denial.sh
source "$SCRIPT_DIR/lib/path-shim-denial.sh"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

path_shim_denial_setup "$WORKDIR" || fail "path-shim-denial setup failed"

# ---------------------------------------------------------------------------
# Stub `shark` (real dry-run delegation, stubbed Shark binary only -- same
# precedent as tc061). Every invocation is appended to shark-calls.log so
# AC-T1's "dispatches nothing else" claim is provable per scenario, not just
# asserted.
# ---------------------------------------------------------------------------
mkdir -p "$WORKDIR/bin"
SHARK_CALLS_LOG="$WORKDIR/shark-calls.log"
: >"$SHARK_CALLS_LOG"
cat >"$WORKDIR/bin/shark" <<SHARK
#!/usr/bin/env bash
set -euo pipefail
echo "\$*" >>"$SHARK_CALLS_LOG"
python3 - "\$@" <<'PY'
import json, os, sys
args = sys.argv[1:]
if args[:2] == ["next", "ROOT-001"]:
    response = json.load(open(os.environ["SHARK_RESPONSE"]))
    path = args[args.index("--prompt-out") + 1]
    open(path, "wb").write(response["prompt"].encode())
    print(json.dumps(response, separators=(",", ":")))
elif args[:2] == ["claim", "TASK-002"]:
    print('{"session_id":"SID-002"}')
elif args and args[0] == "heartbeat":
    print('{"ok":true}')
elif args[:2] == ["status", "advance"]:
    print('{"advanced":true}')
elif args and args[0] == "release":
    print('{"released":true}')
else:
    raise SystemExit("unexpected shark argv: " + repr(args))
PY
SHARK
chmod +x "$WORKDIR/bin/shark"

export PATH="$WORKDIR/bin:$PATH_SHIM_DENIAL_BIN_DIR:$PATH"
export SHARK_RESPONSE="$NEXT_RESPONSE"

mkdir -p "$WORKDIR/scratch-template" "$WORKDIR/retention"

# Only py-bug-due-date-boundary is configured with root_key/scratch_root --
# proving both halves of the Edge Case: a scenario with a fully-replayed
# prelude (this one: entity_family bug, D01-D05 all applicable:false) still
# appears in the matrix with an explicit zero-call line, AND an unconfigured
# scenario is named "skipped" rather than silently omitted or dispatched.
cat >"$WORKDIR/batch-policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
scenarios:
  py-bug-due-date-boundary:
    root_key: "ROOT-001"
    scratch_root: "$WORKDIR/scratch-template"
EOF

# ---------------------------------------------------------------------------
# AC-002 / AC-T1: --mode preview exits 0, denies zero provider/network
# calls, and prints matrix/stages/planned-calls/root/ceilings/pilot-ledger.
# ---------------------------------------------------------------------------
PREVIEW_OUT="$WORKDIR/preview.out"
set +e
"$BATCH" --batch "$WORKDIR/batch-policy.yaml" --retention-root "$WORKDIR/retention" \
	--mode preview >"$PREVIEW_OUT" 2>&1
preview_rc=$?
set -e
[[ "$preview_rc" -eq 0 ]] || fail "preview exited $preview_rc, want 0: $(cat "$PREVIEW_OUT")"

path_shim_denial_assert_empty "preview (--mode preview)" || fail "provider/network invocation during preview"

grep -q "scenario_id=py-bug-due-date-boundary scenario_version=1 family=bug reps=1" "$PREVIEW_OUT" \
	|| fail "preview did not print the scenario matrix row (id/version/family/reps): $(cat "$PREVIEW_OUT")"
grep -q "scenario_id=py-change-priority-scale" "$PREVIEW_OUT" \
	|| fail "preview omitted an unconfigured scenario from the matrix (must still appear, per TC-079 Negative Cases)"

# Edge Case: a scenario whose stages are entirely replayed (here: the whole
# admitted D01-D05 prelude, since entity_family bug skips the product-design
# prelude) must appear with an explicit zero-call line, not be omitted.
grep -q "D01: applicable=False -> replayed (0 planned provider calls)" "$PREVIEW_OUT" \
	|| fail "preview did not print an explicit zero-call line for a fully-replayed scenario: $(cat "$PREVIEW_OUT")"

grep -q "retention root: $WORKDIR/retention" "$PREVIEW_OUT" || fail "preview did not print the resolved retention root"
grep -q "^ceilings:" "$PREVIEW_OUT" || fail "preview did not print declared resource ceilings"
grep -q "pilot-ledger state" "$PREVIEW_OUT" || fail "preview did not print pilot-ledger state per family"
grep -q "bug: no attestation" "$PREVIEW_OUT" || fail "preview did not print the bug family's pilot-ledger state"

# AC-T1: the real dry-run delegation call happened for the ONE configured
# scenario (proving "invokes run-lifecycle.sh --mode dry-run for stage
# resolution"), and "dispatches nothing else" -- the unconfigured scenarios
# produced zero shark invocations, so the ONLY shark calls on record
# originated from the single configured scenario's dry-run resolution.
[[ -s "$SHARK_CALLS_LOG" ]] || fail "expected the real run-lifecycle.sh --mode dry-run call to invoke the stub shark at least once"
grep -q "^next ROOT-001" "$SHARK_CALLS_LOG" || fail "stub shark was never asked to resolve ROOT-001 via 'next' (real dry-run delegation did not happen): $(cat "$SHARK_CALLS_LOG")"
shark_call_count="$(wc -l <"$SHARK_CALLS_LOG" | tr -d ' ')"
# next, claim, status advance, release -> exactly 4 calls for one resolved
# dispatch. More than that would mean a second scenario was also dispatched
# (unconfigured scenarios must never reach shark at all).
[[ "$shark_call_count" -eq 4 ]] || fail "expected exactly 4 shark calls (one resolved dispatch for the one configured scenario), got $shark_call_count: $(cat "$SHARK_CALLS_LOG")"

grep -q "stage resolution: OK -- 1 dispatch(es)" "$PREVIEW_OUT" \
	|| fail "preview did not report the real dry-run stage-resolution outcome: $(cat "$PREVIEW_OUT")"
grep -q "stage resolution: skipped (root_key/scratch_root not configured" "$PREVIEW_OUT" \
	|| fail "preview did not name the unconfigured scenarios as skipped: $(cat "$PREVIEW_OUT")"

echo "TC-079(batch half AC-002/AC-T1): preview exits 0, zero provider/network calls, real dry-run delegation, full-replay scenario not omitted -- PASS"

# ---------------------------------------------------------------------------
# --dry-run alias: REQ-F-001 requires --dry-run to behave identically to
# --mode preview -- the same content, the same zero-call guarantee.
# ---------------------------------------------------------------------------
: >"$SHARK_CALLS_LOG"
: >"$PATH_SHIM_DENIAL_LOG"
DRYRUN_OUT="$WORKDIR/dryrun.out"
set +e
"$BATCH" --batch "$WORKDIR/batch-policy.yaml" --retention-root "$WORKDIR/retention" \
	--dry-run >"$DRYRUN_OUT" 2>&1
dryrun_rc=$?
set -e
[[ "$dryrun_rc" -eq 0 ]] || fail "--dry-run alias exited $dryrun_rc, want 0: $(cat "$DRYRUN_OUT")"
path_shim_denial_assert_empty "--dry-run alias" || fail "provider/network invocation during --dry-run"
grep -q "scenario_id=py-bug-due-date-boundary" "$DRYRUN_OUT" || fail "--dry-run alias did not render the same preview content as --mode preview"

echo "TC-079(REQ-F-001 --dry-run alias): identical zero-spend preview behavior -- PASS"

# ---------------------------------------------------------------------------
# Negative Cases: no third, divergent preview flag convention. --help's
# surface exposes only --mode (preview|pilot|baseline) and --dry-run.
# ---------------------------------------------------------------------------
HELP_OUT="$WORKDIR/help.out"
set +e
"$BATCH" --help >"$HELP_OUT" 2>&1
help_rc=$?
set -e
[[ "$help_rc" -eq 2 ]] || fail "--help exited $help_rc, want 2 (usage banner)"
grep -q -- "--mode preview|pilot|baseline \[--dry-run\]" "$HELP_OUT" \
	|| fail "--help does not document --mode preview|pilot|baseline with --dry-run as its only alias: $(cat "$HELP_OUT")"
# No standalone --preview-style flag distinct from --mode/--dry-run: no
# flag token (a "--"-prefixed word) may itself contain "preview" -- the
# word only legitimately appears as the --mode enum value and in the
# --dry-run alias sentence, neither of which is a "--...preview..." token.
if grep -oE -- '--[a-zA-Z-]*preview[a-zA-Z-]*' "$HELP_OUT"; then
	fail "--help exposes a third preview flag convention beyond --mode/--dry-run: $(cat "$HELP_OUT")"
fi

echo "TC-079(Negative Cases): --help exposes no third preview flag convention -- PASS"

# ---------------------------------------------------------------------------
# This task's own local AC-T2 (task file, beyond TC-079's stated body):
# "A repeat of an already-retained (scenario, rep) is classified and
# skipped; --reclaim-incomplete quarantines rather than deletes."
# Uses RUN_LIFECYCLE_BIN/EVALUATE_LIFECYCLE_BIN stub overrides -- the same
# TD-077 defect-class precedent run-batch.sh's own tests use for
# RUN_ONE_BIN -- since this AC exercises classification/retention
# mechanics, not the real dispatch pipeline TC-079/TC-080 already cover.
# ---------------------------------------------------------------------------
mkdir -p "$WORKDIR/stubbin"
cat >"$WORKDIR/stubbin/run-lifecycle-stub.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
out=""
args=("$@")
for ((i = 0; i < ${#args[@]}; i++)); do
	if [[ "${args[$i]}" == "--output" ]]; then out="${args[$((i + 1))]}"; fi
done
echo '{"outcome":{"terminal":"complete"}}' >"$out"
exit 0
EOF
chmod +x "$WORKDIR/stubbin/run-lifecycle-stub.sh"
cat >"$WORKDIR/stubbin/evaluate-lifecycle-stub.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
out=""
args=("$@")
for ((i = 0; i < ${#args[@]}; i++)); do
	if [[ "${args[$i]}" == "--output" ]]; then out="${args[$((i + 1))]}"; fi
done
echo '{"eligibility":{"aggregate_eligible":true}}' >"$out"
exit 0
EOF
chmod +x "$WORKDIR/stubbin/evaluate-lifecycle-stub.sh"

AC_T2_ROOT="$WORKDIR/ac-t2-retention"
mkdir -p "$AC_T2_ROOT"
cat >"$WORKDIR/ac-t2-policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
scenarios:
  py-bug-due-date-boundary:
    root_key: "ROOT-001"
    scratch_root: "$WORKDIR/scratch-template"
    i05_bundle_dir: "$WORKDIR/scratch-template"
EOF

run_ac_t2_batch() {
	RUN_LIFECYCLE_BIN="$WORKDIR/stubbin/run-lifecycle-stub.sh" \
		EVALUATE_LIFECYCLE_BIN="$WORKDIR/stubbin/evaluate-lifecycle-stub.sh" \
		"$BATCH" --batch "$WORKDIR/ac-t2-policy.yaml" --retention-root "$AC_T2_ROOT" \
		--mode pilot --acknowledge-provider-spend --max-cost-usd 5 \
		--max-wall-clock-seconds 600 --max-generated-tasks 10 \
		--scenarios py-bug-due-date-boundary --reps 1 "$@"
}

set +e
run_ac_t2_batch >"$WORKDIR/ac-t2-first.out" 2>&1
first_rc=$?
set -e
[[ "$first_rc" -eq 0 ]] || fail "AC-T2 first pilot dispatch exited $first_rc, want 0: $(cat "$WORKDIR/ac-t2-first.out")"
[[ -f "$AC_T2_ROOT/scenarios/py-bug-due-date-boundary/1/evaluation.jsonl" ]] || fail "AC-T2 first dispatch did not retain evaluation.jsonl"
grep -q '"classification": "pending_run"' "$AC_T2_ROOT/batch.json" || fail "AC-T2 first dispatch not classified pending_run: $(cat "$AC_T2_ROOT/batch.json")"

set +e
run_ac_t2_batch >"$WORKDIR/ac-t2-second.out" 2>&1
second_rc=$?
set -e
[[ "$second_rc" -eq 0 ]] || fail "AC-T2 repeat of an already-retained pair exited $second_rc, want 0 (skipped_complete)"
grep -q '"classification": "skipped_complete"' "$AC_T2_ROOT/batch.json" || fail "AC-T2 repeat was not classified skipped_complete: $(cat "$AC_T2_ROOT/batch.json")"

echo "TC-079(AC-T2 part 1: repeat of already-retained pair classified and skipped) PASS"

# Simulate a stale prior attempt: quarantine target must be a MOVE of the
# original bytes, never a delete/overwrite.
STALE_DIR="$AC_T2_ROOT/scenarios/py-bug-due-date-boundary/1"
rm -f "$STALE_DIR/evaluation.jsonl"
echo "stale-partial-lifecycle-bytes" >"$STALE_DIR/lifecycle.jsonl"
STALE_DIGEST_BEFORE="$(sha256sum "$STALE_DIR/lifecycle.jsonl" | awk '{print $1}')"

set +e
run_ac_t2_batch >"$WORKDIR/ac-t2-incomplete.out" 2>&1
incomplete_rc=$?
set -e
[[ "$incomplete_rc" -eq 4 ]] || fail "AC-T2 incomplete_prior_attempt without --reclaim-incomplete exited $incomplete_rc, want 4: $(cat "$WORKDIR/ac-t2-incomplete.out")"
[[ -f "$STALE_DIR/lifecycle.jsonl" ]] || fail "AC-T2 stale directory was deleted without --reclaim-incomplete"
STALE_DIGEST_AFTER="$(sha256sum "$STALE_DIR/lifecycle.jsonl" | awk '{print $1}')"
[[ "$STALE_DIGEST_BEFORE" == "$STALE_DIGEST_AFTER" ]] || fail "AC-T2 stale directory bytes changed without --reclaim-incomplete"
grep -q '"classification": "incomplete_prior_attempt"' "$AC_T2_ROOT/batch.json" || fail "AC-T2 stale pair not classified incomplete_prior_attempt"

echo "TC-079(AC-T2 part 2: incomplete prior attempt skipped, bytes untouched, exit 4) PASS"

set +e
run_ac_t2_batch --reclaim-incomplete >"$WORKDIR/ac-t2-reclaim.out" 2>&1
reclaim_rc=$?
set -e
[[ "$reclaim_rc" -eq 0 ]] || fail "AC-T2 --reclaim-incomplete exited $reclaim_rc, want 0: $(cat "$WORKDIR/ac-t2-reclaim.out")"
grep -q '"classification": "quarantined_and_rerun"' "$AC_T2_ROOT/batch.json" || fail "AC-T2 reclaimed pair not classified quarantined_and_rerun: $(cat "$AC_T2_ROOT/batch.json")"

QUARANTINED_FILE="$(find "$AC_T2_ROOT/.incomplete" -name "lifecycle.jsonl" | head -1)"
[[ -n "$QUARANTINED_FILE" ]] || fail "AC-T2 --reclaim-incomplete did not move the stale directory under .incomplete/"
QUARANTINED_DIGEST="$(sha256sum "$QUARANTINED_FILE" | awk '{print $1}')"
[[ "$QUARANTINED_DIGEST" == "$STALE_DIGEST_BEFORE" ]] || fail "AC-T2 quarantined file's bytes do not match the original stale content (a delete+recreate would fail this, a move preserves it)"
[[ -f "$AC_T2_ROOT/scenarios/py-bug-due-date-boundary/1/evaluation.jsonl" ]] || fail "AC-T2 reclaim did not re-dispatch into a fresh target path"

echo "TC-079(AC-T2 part 3: --reclaim-incomplete quarantines via move, never deletes, then reruns) PASS"

echo "TC-079: pass (batch driver half: zero-spend preview, --dry-run alias, no third flag convention, AC-T2 classification/reclaim)"
