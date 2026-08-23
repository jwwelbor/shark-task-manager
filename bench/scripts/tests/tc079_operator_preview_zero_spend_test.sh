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
# UAT-R3-01 rework: writes a real ".oracle.json" sidecar unconditionally
# (evaluate-lifecycle.sh's own naming convention, args.output + ".oracle.json"),
# matching the shape a real evaluate-lifecycle.sh writes when an agent
# fixture checkout IS configured -- AC-T2 below is about classification/
# reclaim mechanics, not about the missing-oracle-source case (that has its
# own dedicated counterfactual block further down), so its happy-path first
# dispatch must retain successfully with genuine sources for everything.
cat >"$WORKDIR/stubbin/evaluate-lifecycle-stub.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
out=""
args=("$@")
for ((i = 0; i < ${#args[@]}; i++)); do
	if [[ "${args[$i]}" == "--output" ]]; then out="${args[$((i + 1))]}"; fi
done
echo '{"eligibility":{"aggregate_eligible":true}}' >"$out"
echo '{"held_back":true,"observed_result":"pass"}' >"${out}.oracle.json"
exit 0
EOF
chmod +x "$WORKDIR/stubbin/evaluate-lifecycle-stub.sh"

# UAT-R3-01 (T-E40-F10-004 rework): retain_pair's entity-history.json now
# requires a REAL source -- the real producer is export-entity-history.sh
# (shark history <root-key> --json against the ephemeral scratch project),
# stubbed here via the SAME ENTITY_HISTORY_EXPORT_BIN sibling-path override
# convention as RUN_LIFECYCLE_BIN/EVALUATE_LIFECYCLE_BIN, writing real
# (non-empty, deterministic) JSON content -- never an empty placeholder --
# so this stub-driven dispatch retains a genuine entity-history.json rather
# than exercising the real `shark` CLI/DB stack this file's own tests are
# not about.
cat >"$WORKDIR/stubbin/entity-history-stub.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
out=""
root_key=""
args=("$@")
for ((i = 0; i < ${#args[@]}; i++)); do
	if [[ "${args[$i]}" == "--output" ]]; then out="${args[$((i + 1))]}"; fi
	if [[ "${args[$i]}" == "--root-key" ]]; then root_key="${args[$((i + 1))]}"; fi
done
python3 -c 'import json,sys; obj={"entity_type":"task","entity_key":sys.argv[1],"history":[{"timestamp":"2026-01-01T00:00:00Z","old_status":"todo","new_status":"in_progress","agent":"stub"}],"total":1}; open(sys.argv[2],"w",encoding="utf-8").write(json.dumps(obj, sort_keys=True, separators=(",", ":")) + "\n")' "$root_key" "$out"
EOF
chmod +x "$WORKDIR/stubbin/entity-history-stub.sh"

AC_T2_ROOT="$WORKDIR/ac-t2-retention"
mkdir -p "$AC_T2_ROOT"
# UAT-R3-01 rework: a real i05_bundle_dir (evidence content plus a genuine
# transcripts/ subtree) -- AC-T2 below is about classification/reclaim
# mechanics and must retain successfully on its happy-path first dispatch,
# so every required artifact needs a real source (retain_pair refuses
# closed otherwise). The missing-source case gets its own dedicated
# counterfactual block further down using a thin bundle deliberately.
AC_T2_I05_BUNDLE="$WORKDIR/ac-t2-i05-bundle"
mkdir -p "$AC_T2_I05_BUNDLE/transcripts"
echo '{"stage":"develop"}' >"$AC_T2_I05_BUNDLE/bundle.json"
echo "ac-t2 fixture transcript" >"$AC_T2_I05_BUNDLE/transcripts/develop.log"
cat >"$WORKDIR/ac-t2-policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
scenarios:
  py-bug-due-date-boundary:
    root_key: "ROOT-001"
    scratch_root: "$WORKDIR/scratch-template"
    i05_bundle_dir: "$AC_T2_I05_BUNDLE"
EOF

run_ac_t2_batch() {
	RUN_LIFECYCLE_BIN="$WORKDIR/stubbin/run-lifecycle-stub.sh" \
		EVALUATE_LIFECYCLE_BIN="$WORKDIR/stubbin/evaluate-lifecycle-stub.sh" \
		ENTITY_HISTORY_EXPORT_BIN="$WORKDIR/stubbin/entity-history-stub.sh" \
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

# ---------------------------------------------------------------------------
# UAT-R3-01 (round 3) positive check: every required artifact AC-T2's
# happy-path first dispatch retained above has a REAL, non-empty
# source_path and a digest that matches a fresh recompute -- superseding
# the pre-fix "honest-empty digest" assertion this block used to make
# (fabricated placeholders with source_path=="" are no longer produced or
# accepted anywhere in this codebase; see the missing-source counterfactual
# immediately below for the refusal half of this fix).
# ---------------------------------------------------------------------------
AC_T2_REP_DIR="$AC_T2_ROOT/scenarios/py-bug-due-date-boundary/1"
set +e
python3 - "$AC_T2_REP_DIR" "$REPO_ROOT/bench/reports/lifecycle-baseline-schema.yaml" <<'PY'
import hashlib
import json
import os
import sys

import yaml

rep_dir, schema_path = sys.argv[1:3]

with open(schema_path, encoding="utf-8") as f:
    schema = yaml.safe_load(f) or {}
required = [n for n in (schema.get("retention_required_artifacts") or []) if n != "manifest.json"]
if not required:
    print("schema retention_required_artifacts is missing or empty", file=sys.stderr)
    raise SystemExit(1)

with open(f"{rep_dir}/manifest.json", encoding="utf-8") as f:
    manifest = json.load(f)
artifacts = manifest["artifacts"]


def digest_of_path(path):
    if os.path.isdir(path):
        entries = []
        for root, dirs, files in os.walk(path):
            dirs.sort()
            for fname in sorted(files):
                fpath = os.path.join(root, fname)
                relpath = os.path.relpath(fpath, path).replace(os.sep, "/")
                with open(fpath, "rb") as fh:
                    entries.append({"path": relpath, "sha256": hashlib.sha256(fh.read()).hexdigest()})
        entries.sort(key=lambda e: e["path"])
        canonical = json.dumps(entries, sort_keys=True, separators=(",", ":")).encode("utf-8")
        return hashlib.sha256(canonical).hexdigest()
    with open(path, "rb") as fh:
        return hashlib.sha256(fh.read()).hexdigest()


for name in required:
    entry = artifacts.get(name)
    if entry is None:
        print(f"manifest.json missing artifacts.{name}", file=sys.stderr)
        raise SystemExit(1)
    if not entry.get("source_path"):
        print(f"manifest.json artifacts.{name}.source_path is empty -- UAT-R3-01: a successfully retained required artifact must always carry a real source", file=sys.stderr)
        raise SystemExit(1)
    recomputed = digest_of_path(os.path.join(rep_dir, name))
    if recomputed != entry.get("sha256"):
        print(f"manifest.json artifacts.{name}.sha256={entry.get('sha256')!r} does not match recomputed digest {recomputed!r}", file=sys.stderr)
        raise SystemExit(1)

print("every required artifact carries a real, non-empty source_path and a byte-preserved digest")
PY
real_source_rc=$?
set -e
[[ "$real_source_rc" -eq 0 ]] || fail "real-source artifact digest verification failed (UAT-R3-01)"

echo "TC-079(rework: every required artifact carries a real, non-empty source_path -- UAT-R3-01) PASS"

# ---------------------------------------------------------------------------
# UAT-R3-01 (round 3) counterfactual: when a required artifact's source is
# genuinely absent (i05_bundle_dir has no transcripts/ subtree, AND the
# evaluate stub never writes a ".oracle.json" sidecar -- exactly the two
# "honest empty" shapes the pre-fix fabrication used to accept as retained),
# the WHOLE pair MUST be refused -- not retained with a fabricated
# placeholder. This test fails against the pre-fix fabrication behavior
# (which retained the pair successfully with source_path=="" placeholders)
# and passes only now that retain_pair/dispatch_pair refuse closed.
# ---------------------------------------------------------------------------
cat >"$WORKDIR/stubbin/evaluate-lifecycle-no-oracle-stub.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
out=""
args=("$@")
for ((i = 0; i < ${#args[@]}; i++)); do
	if [[ "${args[$i]}" == "--output" ]]; then out="${args[$((i + 1))]}"; fi
done
echo '{"eligibility":{"aggregate_eligible":true}}' >"$out"
# Deliberately never writes "${out}.oracle.json" -- the real
# evaluate-lifecycle.sh shape when no agent fixture checkout is configured.
exit 0
EOF
chmod +x "$WORKDIR/stubbin/evaluate-lifecycle-no-oracle-stub.sh"

MISSING_SOURCE_ROOT="$WORKDIR/missing-source-retention"
mkdir -p "$MISSING_SOURCE_ROOT"
cat >"$WORKDIR/missing-source-policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
scenarios:
  py-bug-due-date-boundary:
    root_key: "ROOT-001"
    scratch_root: "$WORKDIR/scratch-template"
    i05_bundle_dir: "$WORKDIR/scratch-template"
EOF

set +e
RUN_LIFECYCLE_BIN="$WORKDIR/stubbin/run-lifecycle-stub.sh" \
	EVALUATE_LIFECYCLE_BIN="$WORKDIR/stubbin/evaluate-lifecycle-no-oracle-stub.sh" \
	ENTITY_HISTORY_EXPORT_BIN="$WORKDIR/stubbin/entity-history-stub.sh" \
	"$BATCH" --batch "$WORKDIR/missing-source-policy.yaml" --retention-root "$MISSING_SOURCE_ROOT" \
	--mode pilot --acknowledge-provider-spend --max-cost-usd 5 \
	--max-wall-clock-seconds 600 --max-generated-tasks 10 \
	--scenarios py-bug-due-date-boundary --reps 1 >"$WORKDIR/missing-source.out" 2>&1
missing_source_rc=$?
set -e
[[ "$missing_source_rc" -eq 4 ]] || fail "missing-source dispatch expected exit 4 (batch completed with a failed pair), got $missing_source_rc: $(cat "$WORKDIR/missing-source.out")"
[[ ! -f "$MISSING_SOURCE_ROOT/scenarios/py-bug-due-date-boundary/1/manifest.json" ]] || fail "missing-source dispatch retained manifest.json -- expected the whole pair refused, never a fabricated placeholder"
grep -q '"classification": "failed"' "$MISSING_SOURCE_ROOT/batch.json" || fail "missing-source dispatch not classified failed: $(cat "$MISSING_SOURCE_ROOT/batch.json")"
grep -q "required_artifact_source_unavailable" "$MISSING_SOURCE_ROOT/invalid/index.jsonl" || fail "missing-source dispatch invalid/index.jsonl missing required_artifact_source_unavailable: $(cat "$MISSING_SOURCE_ROOT/invalid/index.jsonl")"

echo "TC-079(UAT-R3-01 counterfactual: batch driver refuses the whole pair, no fabricated placeholder, when required transcripts/oracle sources are absent) PASS"

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

# ---------------------------------------------------------------------------
# T-E40-F10-004 rework (code-review-2026-08-20T1731-E40-F10.md finding 1,
# REQ-F-004/AC-005/AC-006): retain_pair() must copy REAL byte content from
# i05_bundle_dir into evidence/ and transcripts/, and real oracle.json
# content when evaluate-lifecycle.sh produces one -- not empty placeholder
# directories/files that satisfy every downstream presence/digest check
# without holding any real content. Reuses the same
# RUN_LIFECYCLE_BIN/EVALUATE_LIFECYCLE_BIN stub-override precedent as AC-T2
# above, since this proves retain_pair's own copying behavior, not the real
# dispatch pipeline TC-079/TC-080 already cover.
# ---------------------------------------------------------------------------
I05_BUNDLE="$WORKDIR/i05-bundle"
mkdir -p "$I05_BUNDLE/stages" "$I05_BUNDLE/transcripts"
echo '{"schema_version":"1.0","stages":[{"stage_key":"develop"}]}' >"$I05_BUNDLE/bundle.json"
echo '{"stage_key":"develop"}' >"$I05_BUNDLE/stages/1-develop.json"
echo "tc079-fixture-transcript-bytes" >"$I05_BUNDLE/transcripts/develop.log"

cat >"$WORKDIR/stubbin/evaluate-lifecycle-oracle-stub.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
out=""
args=("$@")
for ((i = 0; i < ${#args[@]}; i++)); do
	if [[ "${args[$i]}" == "--output" ]]; then out="${args[$((i + 1))]}"; fi
done
echo '{"eligibility":{"aggregate_eligible":true}}' >"$out"
echo '{"held_back":true,"observed_result":"pass"}' >"${out}.oracle.json"
exit 0
EOF
chmod +x "$WORKDIR/stubbin/evaluate-lifecycle-oracle-stub.sh"

RETAIN_ROOT="$WORKDIR/retain-content-check"
mkdir -p "$RETAIN_ROOT"
cat >"$WORKDIR/retain-content-policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
scenarios:
  py-bug-due-date-boundary:
    root_key: "ROOT-001"
    scratch_root: "$WORKDIR/scratch-template"
    i05_bundle_dir: "$I05_BUNDLE"
EOF

set +e
RUN_LIFECYCLE_BIN="$WORKDIR/stubbin/run-lifecycle-stub.sh" \
	EVALUATE_LIFECYCLE_BIN="$WORKDIR/stubbin/evaluate-lifecycle-oracle-stub.sh" \
	ENTITY_HISTORY_EXPORT_BIN="$WORKDIR/stubbin/entity-history-stub.sh" \
	"$BATCH" --batch "$WORKDIR/retain-content-policy.yaml" --retention-root "$RETAIN_ROOT" \
	--mode pilot --acknowledge-provider-spend --max-cost-usd 5 \
	--max-wall-clock-seconds 600 --max-generated-tasks 10 \
	--scenarios py-bug-due-date-boundary --reps 1 >"$WORKDIR/retain-content.out" 2>&1
retain_content_rc=$?
set -e
[[ "$retain_content_rc" -eq 0 ]] || fail "retention-content dispatch exited $retain_content_rc, want 0: $(cat "$WORKDIR/retain-content.out")"

REP_DIR="$RETAIN_ROOT/scenarios/py-bug-due-date-boundary/1"
[[ -f "$REP_DIR/evidence/bundle.json" ]] || fail "retain_pair did not copy bundle.json into evidence/"
diff -q "$I05_BUNDLE/bundle.json" "$REP_DIR/evidence/bundle.json" >/dev/null || fail "evidence/bundle.json bytes do not match i05_bundle_dir source"
[[ -f "$REP_DIR/evidence/stages/1-develop.json" ]] || fail "retain_pair did not copy stages/1-develop.json into evidence/"
diff -q "$I05_BUNDLE/stages/1-develop.json" "$REP_DIR/evidence/stages/1-develop.json" >/dev/null || fail "evidence/stages/1-develop.json bytes do not match source"
[[ -f "$REP_DIR/transcripts/develop.log" ]] || fail "retain_pair did not copy transcripts/develop.log into transcripts/"
diff -q "$I05_BUNDLE/transcripts/develop.log" "$REP_DIR/transcripts/develop.log" >/dev/null || fail "transcripts/develop.log bytes do not match source"
[[ ! -e "$REP_DIR/evidence/transcripts" ]] || fail "evidence/ must not duplicate the transcripts/ subtree"

[[ -s "$REP_DIR/oracle.json" ]] || fail "retain_pair did not copy real oracle.json content when evaluate-lifecycle.sh produced one"
grep -q '"held_back":true' "$REP_DIR/oracle.json" || fail "oracle.json content is not the real evaluator output"

# manifest.json must record real, non-placeholder digests for evidence/
# transcripts/oracle.json -- recomputed independently here (not reusing
# retain_pair's own digest function) so a passing assertion actually
# proves byte preservation, not merely that SOME digest was written.
set +e
python3 - "$REP_DIR" <<'PY'
import hashlib, json, os, sys

rep_dir = sys.argv[1]


def digest_of_path(path):
    if os.path.isdir(path):
        entries = []
        for root, dirs, files in os.walk(path):
            dirs.sort()
            for fname in sorted(files):
                fpath = os.path.join(root, fname)
                relpath = os.path.relpath(fpath, path).replace(os.sep, "/")
                with open(fpath, "rb") as fh:
                    entries.append({"path": relpath, "sha256": hashlib.sha256(fh.read()).hexdigest()})
        entries.sort(key=lambda e: e["path"])
        canonical = json.dumps(entries, sort_keys=True, separators=(",", ":")).encode("utf-8")
        return hashlib.sha256(canonical).hexdigest()
    with open(path, "rb") as fh:
        return hashlib.sha256(fh.read()).hexdigest()


with open(os.path.join(rep_dir, "manifest.json")) as f:
    manifest = json.load(f)
artifacts = manifest["artifacts"]
empty_dir_digest = hashlib.sha256(b"[]").hexdigest()
for name in ("evidence", "transcripts", "oracle.json"):
    if name not in artifacts:
        print(f"manifest.json missing artifacts.{name}", file=sys.stderr)
        raise SystemExit(1)
    recorded = artifacts[name]["sha256"]
    actual = digest_of_path(os.path.join(rep_dir, name))
    if recorded != actual:
        print(f"manifest.json artifacts.{name}.sha256={recorded!r} does not match freshly recomputed digest {actual!r}", file=sys.stderr)
        raise SystemExit(1)
    if not artifacts[name]["source_path"]:
        print(f"manifest.json artifacts.{name}.source_path is empty despite real source content", file=sys.stderr)
        raise SystemExit(1)
    if recorded == empty_dir_digest:
        print(f"manifest.json artifacts.{name}.sha256 is the empty-directory placeholder digest, not real content", file=sys.stderr)
        raise SystemExit(1)
print("manifest digests verified real")
PY
manifest_check_rc=$?
set -e
[[ "$manifest_check_rc" -eq 0 ]] || fail "manifest.json digest verification failed for evidence/transcripts/oracle.json"

echo "TC-079(rework: retain_pair copies real evidence/transcripts/oracle.json content, digests verified) PASS"

echo "TC-079: pass (batch driver half: zero-spend preview, --dry-run alias, no third flag convention, AC-T2 classification/reclaim, real evidence/transcripts/oracle.json retention)"

# ===========================================================================
# TC-079 (comparison half) / T-E40-F10-005: zero-provider-call operator
# preview for run-review-comparison.sh (spec.md REQ-F-001, REQ-F-014,
# REQ-NF-002; test-plan.md TC-079). Caller-Path Contract: only the
# enumerated PATH-shim provider/network denial process is a stub; the
# `run-lifecycle.sh --mode dry-run` invocation itself is real (same
# precedent as the batch half above -- the `shark` executable it calls is
# stubbed, not the preview-content assembly, stage resolution, or the
# dry-run delegation call).
# ===========================================================================
COMPARISON="$SCRIPTS_DIR/run-review-comparison.sh"
[[ -x "$COMPARISON" ]] || fail "bench/scripts/run-review-comparison.sh missing or not executable"

CMP_WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR" "$CMP_WORKDIR"' EXIT

path_shim_denial_setup "$CMP_WORKDIR" || fail "path-shim-denial setup failed (comparison half)"

mkdir -p "$CMP_WORKDIR/bin"
CMP_SHARK_CALLS_LOG="$CMP_WORKDIR/shark-calls.log"
: >"$CMP_SHARK_CALLS_LOG"
cat >"$CMP_WORKDIR/bin/shark" <<SHARK
#!/usr/bin/env bash
set -euo pipefail
echo "\$*" >>"$CMP_SHARK_CALLS_LOG"
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
chmod +x "$CMP_WORKDIR/bin/shark"

export PATH="$CMP_WORKDIR/bin:$PATH_SHIM_DENIAL_BIN_DIR:$PATH"
export SHARK_RESPONSE="$NEXT_RESPONSE"

mkdir -p "$CMP_WORKDIR/scratch-template" "$CMP_WORKDIR/truth-set" "$CMP_WORKDIR/retention"
echo "seeded truth-set marker" >"$CMP_WORKDIR/truth-set/marker.txt"

cat >"$CMP_WORKDIR/candidate.yaml" <<EOF
schema_version: "1.0"
scenario_id: "py-bug-due-date-boundary"
gates:
  qa:
    root_key: "ROOT-001"
    scratch_root: "$CMP_WORKDIR/scratch-template"
  deep_review:
    root_key: "ROOT-001"
    scratch_root: "$CMP_WORKDIR/scratch-template"
    i05_bundle_dir: "$CMP_WORKDIR/truth-set"
EOF

run_comparison_preview() {
	# run_comparison_preview <comparison-mode> <out-file>
	local cmode="$1" out="$2"
	set +e
	"$COMPARISON" --candidate "$CMP_WORKDIR/candidate.yaml" --retention-root "$CMP_WORKDIR/retention" \
		--mode preview --comparison-mode "$cmode" >"$out" 2>&1
	local rc=$?
	set -e
	return "$rc"
}

# AC-002 / AC-T1: both comparison-mode previews exit 0, deny zero
# provider/network calls, and print candidate identities, workflow-policy
# identities, expected provider-call inventory, truth-set availability,
# and fix rules -- before any spend.
CMP_PREVIEW_OUT="$CMP_WORKDIR/preview-independent.out"
set +e
run_comparison_preview "independent_frozen_candidate" "$CMP_PREVIEW_OUT"
cmp_preview_rc=$?
set -e
[[ "$cmp_preview_rc" -eq 0 ]] || fail "comparison preview (independent_frozen_candidate) exited $cmp_preview_rc, want 0: $(cat "$CMP_PREVIEW_OUT")"

path_shim_denial_assert_empty "comparison preview (independent_frozen_candidate)" || fail "provider/network invocation during comparison preview"

grep -q "comparison mode: independent_frozen_candidate" "$CMP_PREVIEW_OUT" || fail "comparison preview did not print the comparison mode: $(cat "$CMP_PREVIEW_OUT")"
grep -q "^gate: qa" "$CMP_PREVIEW_OUT" || fail "comparison preview did not print the qa gate"
grep -q "^gate: deep_review" "$CMP_PREVIEW_OUT" || fail "comparison preview did not print the deep_review gate"
grep -q "expected provider-call inventory" "$CMP_PREVIEW_OUT" || fail "comparison preview did not print the expected provider-call inventory"
grep -q "truth-set availability: False" "$CMP_PREVIEW_OUT" || fail "comparison preview did not print truth-set availability: False for the qa gate (no i05_bundle_dir configured)"
grep -q "truth-set availability: True" "$CMP_PREVIEW_OUT" || fail "comparison preview did not print truth-set availability: True for the deep_review gate (i05_bundle_dir configured and non-empty)"
grep -q "candidate identity" "$CMP_PREVIEW_OUT" || fail "comparison preview did not print candidate identities"
grep -q "workflow-policy identity:" "$CMP_PREVIEW_OUT" || fail "comparison preview did not print workflow-policy identities"
grep -q "fix rules: fixes_allowed_between_gates=" "$CMP_PREVIEW_OUT" || fail "comparison preview did not print fix rules (whether fixes are permitted between gates)"
grep -q "identity:" "$CMP_PREVIEW_OUT" || fail "comparison preview did not print an identity block"

[[ -s "$CMP_SHARK_CALLS_LOG" ]] || fail "expected the real run-lifecycle.sh --mode dry-run call to invoke the stub shark at least once"
cmp_next_calls="$(grep -c "^next ROOT-001" "$CMP_SHARK_CALLS_LOG")"
[[ "$cmp_next_calls" -eq 2 ]] || fail "expected exactly 2 real dry-run resolutions (one per gate), got $cmp_next_calls: $(cat "$CMP_SHARK_CALLS_LOG")"

echo "TC-079(comparison half AC-002/AC-T1, independent_frozen_candidate): preview exits 0, zero provider/network calls, real dry-run delegation per gate, candidates/policies/provider-call-inventory/truth-set/fix-rules all printed -- PASS"

# Same assertions again with --comparison-mode sequential_delivery: the
# preview content and zero-spend guarantee do not depend on comparison
# mode (REQ-F-014 requires preview support for both architecture modes).
: >"$CMP_SHARK_CALLS_LOG"
: >"$PATH_SHIM_DENIAL_LOG"
CMP_PREVIEW_SEQ_OUT="$CMP_WORKDIR/preview-sequential.out"
set +e
run_comparison_preview "sequential_delivery" "$CMP_PREVIEW_SEQ_OUT"
cmp_preview_seq_rc=$?
set -e
[[ "$cmp_preview_seq_rc" -eq 0 ]] || fail "comparison preview (sequential_delivery) exited $cmp_preview_seq_rc, want 0: $(cat "$CMP_PREVIEW_SEQ_OUT")"
path_shim_denial_assert_empty "comparison preview (sequential_delivery)" || fail "provider/network invocation during comparison preview (sequential_delivery)"
grep -q "comparison mode: sequential_delivery" "$CMP_PREVIEW_SEQ_OUT" || fail "comparison preview did not print the sequential_delivery comparison mode: $(cat "$CMP_PREVIEW_SEQ_OUT")"

echo "TC-079(comparison half, sequential_delivery): identical zero-spend preview behavior -- PASS"

# ---------------------------------------------------------------------------
# AC-T3 (this task's own local AC, T-E40-F10-014's future static-grep
# target): run-review-comparison.sh must contain no field-by-field
# candidate/policy identity comparison logic of its own -- proven here by
# asserting the comparator's own candidate/policy digest field-name
# vocabulary never appears in this driver's source at all.
# ---------------------------------------------------------------------------
for forbidden_field in base_commit tree_digest binary_diff_digest changed_path_digest \
	dirty_untracked_manifest test_suite_digest identity_digest prompt_digest \
	rendered_prompt_digest review_bundle_digest deep_review_bundle_digest \
	workflow_policy_identity_digest; do
	if grep -q -- "$forbidden_field" "$COMPARISON"; then
		fail "AC-T3: run-review-comparison.sh contains the comparator's own identity field name '$forbidden_field' (must delegate identity adjudication entirely, never re-implement it)"
	fi
done

echo "TC-079(AC-T3): run-review-comparison.sh contains no comparator identity field-name vocabulary -- PASS"

echo "TC-079: pass (comparison driver half: zero-spend preview in both comparison modes, candidates/policies/provider-call-inventory/truth-set/fix-rules printed, no local identity vocabulary)"
