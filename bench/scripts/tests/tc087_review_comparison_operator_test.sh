#!/usr/bin/env bash
# TC-087 / T-E40-F10-005: operator review-comparison operation (both
# architecture modes) (spec.md REQ-F-014, REQ-F-015; test-plan.md TC-087
# full body). AC-010, AC-T2, AC-T3.
#
# Caller-Path Contract (test-plan.md table): `run-review-comparison.sh`,
# invoked with its real CLI, which internally invokes the REAL, unstubbed
# `compare-lifecycle-evaluations.sh --left <e1> --right <e2> --mode <mode>
# --output <comparison.json>`. Do not mock the comparator call, its
# accept/reject verdict, or divergence-reason preservation.
#
# This test proves that contract with ZERO stub seams anywhere in the
# gate-dispatch path: the two I-08 evaluation records TC-087's Preconditions
# call "retained" are pre-populated directly at the exact retention path
# run-review-comparison.sh's own skipped_complete classification looks for
# (mirroring run-lifecycle-batch.sh's classify_pair/skipped_complete reuse,
# T-E40-F10-004 precedent) -- dispatch is never reached, RUN_LIFECYCLE_BIN
# and EVALUATE_LIFECYCLE_BIN are never invoked, and the ONLY real subprocess
# this test exercises is the real, unmodified comparator itself.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPARISON="$SCRIPTS_DIR/run-review-comparison.sh"
COMPARATOR="$SCRIPTS_DIR/compare-lifecycle-evaluations.sh"
SCENARIO_ID="py-bug-due-date-boundary"
# Mirrors run-review-comparison.sh's gate_rep() (GATE_REP_BASE=900000,
# qa=BASE+1, deep_review=BASE+2) -- code-review-2026-08-21T0330-E40-F10.md
# finding 1 moved gate reps out of the low 1/2 range run-lifecycle-batch.sh
# allocates from into a reserved band so the two producers' rep
# allocations cannot collide regardless of dispatch order.
QA_REP=900001
DR_REP=900002

fail() {
	echo "TC-087 FAIL: $1" >&2
	exit 1
}

[[ -x "$COMPARISON" ]] || fail "bench/scripts/run-review-comparison.sh missing or not executable"
[[ -x "$COMPARATOR" ]] || fail "bench/scripts/compare-lifecycle-evaluations.sh missing or not executable"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# ---------------------------------------------------------------------------
# Fixture builder: writes independent/sequential/divergent I-08 evaluation
# record pairs (same shape tc072_review_comparison_modes_test.sh already
# uses to exercise compare-lifecycle-evaluations.sh directly), one JSON file
# per side, under $1.
# ---------------------------------------------------------------------------
python3 - "$WORKDIR" <<'PY'
import copy, hashlib, json, pathlib, sys
root = pathlib.Path(sys.argv[1])
digest = "a" * 64

def candidate_digest_fields(candidate):
    return {key: candidate[key] for key in ("base_commit", "tree_digest", "binary_diff_digest", "changed_path_digest", "dirty_untracked_manifest", "test_suite_digest")}

def with_identity_digest(candidate):
    candidate = dict(candidate)
    candidate["identity_digest"] = hashlib.sha256(json.dumps(candidate_digest_fields(candidate), sort_keys=True, separators=(",", ":")).encode()).hexdigest()
    return candidate

identity = {key: value for key, value in {
    "scenario_id": "s", "scenario_version": "1", "fixture_id": "f", "fixture_digest": digest,
    "adapter_id": "a", "adapter_version": "1", "toolchain_identity": [{"key": "go", "value": "1"}],
    "shark_binary_digest": digest, "shark_content_digest": digest, "rendered_prompt_digest": digest,
    "rendered_prompt_digests": [digest], "provider": "fixture", "model": "m", "effort": "low",
    "provider_identity": [{"stage": "qa", "provider": "fixture", "model": "m", "effort": "low"}],
    "judge_digest": digest, "judge_identity": {"model": "j", "configuration": "c"},
    "reference_digest": digest, "reference_digests": [digest], "resource_policy_digest": digest,
}.items()}

base_candidate = with_identity_digest({
    "base_commit": "b" * 40, "tree_digest": digest, "binary_diff_digest": digest,
    "changed_path_digest": digest, "dirty_untracked_manifest": digest,
    "test_suite_digest": digest, "snapshot_digest": digest,
})

def make_policy(**overrides):
    policy = {
        "enabled_gates": ["qa", "deep_review"], "gate_order": ["qa", "deep_review"],
        "reviewer": {"provider": "fixture", "model": "m", "effort": "low"},
        "prompt_digest": digest, "rendered_prompt_digest": digest,
        "review_bundle_digest": digest, "deep_review_bundle_digest": digest,
        "fixes_allowed_between_gates": False, "fix_policy": "none",
    }
    policy.update(overrides)
    policy["workflow_policy_identity_digest"] = hashlib.sha256(json.dumps({k: v for k, v in policy.items() if k != "workflow_policy_identity_digest"}, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
    return policy

base_policy = make_policy()

def finding(fid, gate, confirmed=True):
    return {"f09_finding_id": fid, "gate": gate, "final_disposition": "confirmed" if confirmed else "unconfirmed", "confirmation_source": "seeded_truth_set" if confirmed else "none"}

def base_record(evaluation_id, candidate_snapshots, policy=None, findings=None):
    return {
        "evaluation_id": evaluation_id,
        "identity": identity,
        "workflow_policy": policy or base_policy,
        "eligibility": {"aggregate_eligible": True},
        "candidate_snapshots": candidate_snapshots,
        "review_findings": {"normalized_findings": findings or [finding("f09-1", "qa")], "derived_counts": {"confirmed": 1}},
    }

def write(name, record):
    path = root / f"{name}.jsonl"
    path.write_text(json.dumps(record) + "\n")
    return path

# --- Pair A: identity-compatible independent_frozen_candidate pair -------
qa_compatible = base_record("qa-compatible", [{"stage": "qa", "candidate": base_candidate}])
dr_compatible = base_record("dr-compatible", [{"stage": "deep_review", "candidate": base_candidate}])
write("indep-compatible-qa", qa_compatible)
write("indep-compatible-dr", dr_compatible)

# --- Pair B: one-field-divergent independent_frozen_candidate pair -------
divergent_candidate = with_identity_digest(dict(base_candidate, tree_digest="c" * 64))
dr_divergent = base_record("dr-divergent", [{"stage": "deep_review", "candidate": divergent_candidate}])
write("indep-divergent-qa", qa_compatible)
write("indep-divergent-dr", dr_divergent)

# --- Pair C: branch-name/HEAD-only candidate identity (must be rejected
#     by the DELEGATED comparator's own malformed-digest check, never a
#     local F10 special-case) ------------------------------------------
branch_candidate = with_identity_digest(dict(base_candidate, base_commit="HEAD"))
dr_branch = base_record("dr-branch", [{"stage": "deep_review", "candidate": branch_candidate}])
write("indep-branch-qa", qa_compatible)
write("indep-branch-dr", dr_branch)

# --- Pair D: workflow-policy-only divergence (candidate identical) -------
policy_divergent = make_policy(fixes_allowed_between_gates=True)
dr_policy_divergent = base_record("dr-policy-divergent", [{"stage": "deep_review", "candidate": base_candidate}], policy=policy_divergent)
write("indep-policy-divergent-qa", qa_compatible)
write("indep-policy-divergent-dr", dr_policy_divergent)

# --- Pair E: three-candidate sequential_delivery chain, exercised on
#     both sides (qa view of the chain vs deep_review view of the SAME
#     chain) -- every intervening candidate is present on both sides. ---
chain_candidates = [with_identity_digest(dict(base_candidate, tree_digest=c * 64)) for c in ("1", "2", "3")]
chain_snapshots = [
    {"stage": "qa", "gate": "qa", "candidate": chain_candidates[0]},
    {"stage": "qa", "gate": "qa", "candidate": chain_candidates[1]},
    {"stage": "deep_review", "gate": "deep_review", "candidate": chain_candidates[2]},
]
qa_chain = base_record("qa-chain", chain_snapshots, findings=[finding("f09-1", "qa"), finding("f09-2", "qa")])
dr_chain = base_record("dr-chain", chain_snapshots, findings=[finding("f09-1", "qa"), finding("f09-2", "qa"), finding("f09-3", "deep_review")])
write("seq-chain-qa", qa_chain)
write("seq-chain-dr", dr_chain)

print("fixtures written")
PY

[[ $? -eq 0 ]] || fail "fixture builder failed"

# ---------------------------------------------------------------------------
# retain_pair_fixtures <root> <qa_fixture> <dr_fixture>
# Pre-populates the exact retention path run-review-comparison.sh's own
# skipped_complete classification looks for -- so dispatch is never
# reached and RUN_LIFECYCLE_BIN/EVALUATE_LIFECYCLE_BIN are never invoked.
#
# T-E40-F10-005 rework (code-review-2026-08-20T2138-E40-F10.md finding 2):
# the driver's retention layout is now the SAME (scenario_id, rep) pair
# shape run-lifecycle-batch.sh uses -- scenarios/<scenario_id>/<rep>/ with
# a manifest.json carrying scenario_id/rep/gate -- not a gate-named
# directory. gate_rep() in run-review-comparison.sh maps qa/deep_review to
# QA_REP/DR_REP (round-3 rework: a reserved band, not the low 1/2 range --
# code-review-2026-08-21T0330-E40-F10.md finding 1). The fixture below
# writes minimal-but-real manifest.json files at those two paths so
# gate_dest_provenance_ok()'s provenance check (manifest.json's "gate"
# field) accepts them as already-retained by THIS driver for THIS gate,
# not an unrelated batch pair -- proving the skip path against the real
# provenance check, not just against a bare evaluation.jsonl existence
# check.
# ---------------------------------------------------------------------------
retain_pair_fixtures() {
	local root="$1" qa_fixture="$2" dr_fixture="$3"
	local qa_dir="$root/scenarios/$SCENARIO_ID/$QA_REP" dr_dir="$root/scenarios/$SCENARIO_ID/$DR_REP"
	mkdir -p "$qa_dir" "$dr_dir"
	cp "$WORKDIR/${qa_fixture}.jsonl" "$qa_dir/evaluation.jsonl"
	cp "$WORKDIR/${dr_fixture}.jsonl" "$dr_dir/evaluation.jsonl"
	python3 -c 'import json,sys
scenario_id, rep, gate, dest = sys.argv[1:5]
manifest = {"scenario_id": scenario_id, "rep": int(rep), "gate": gate, "artifacts": {}}
open(dest, "w").write(json.dumps(manifest, sort_keys=True, separators=(",", ":")) + "\n")' "$SCENARIO_ID" "$QA_REP" qa "$qa_dir/manifest.json"
	python3 -c 'import json,sys
scenario_id, rep, gate, dest = sys.argv[1:5]
manifest = {"scenario_id": scenario_id, "rep": int(rep), "gate": gate, "artifacts": {}}
open(dest, "w").write(json.dumps(manifest, sort_keys=True, separators=(",", ":")) + "\n")' "$SCENARIO_ID" "$DR_REP" deep_review "$dr_dir/manifest.json"
}

CANDIDATE_YAML="$WORKDIR/candidate.yaml"
cat >"$CANDIDATE_YAML" <<EOF
schema_version: "1.0"
scenario_id: "$SCENARIO_ID"
gates:
  qa:
    root_key: ""
    scratch_root: ""
  deep_review:
    root_key: ""
    scratch_root: ""
EOF

ACK_FLAGS=(--acknowledge-provider-spend --max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10)

run_case() {
	# run_case <label> <root> <comparison_mode>
	# Writes stdout to <root>.stdout (the driver's published/verbatim JSON
	# output only) and stderr to <root>.stderr (diagnostic banners), and
	# a combined <root>.out for convenience greps. Prints the exit code.
	local label="$1" root="$2" cmode="$3"
	local stdout_file="$root.stdout" stderr_file="$root.stderr" out="$root.out"
	set +e
	"$COMPARISON" --candidate "$CANDIDATE_YAML" --retention-root "$root" \
		--mode pilot --comparison-mode "$cmode" "${ACK_FLAGS[@]}" >"$stdout_file" 2>"$stderr_file"
	local rc=$?
	set -e
	cat "$stderr_file" "$stdout_file" >"$out"
	echo "$rc"
}

# ===========================================================================
# Case A: identity-compatible pair, independent_frozen_candidate -> ACCEPT
# and PUBLISH. Cross-checked against a direct, real comparator invocation
# on the identical two files so the driver's stdout is proven byte-for-byte
# identical to the comparator's own output (not just "similar").
# ===========================================================================
ROOT_A="$WORKDIR/root-a"
mkdir -p "$ROOT_A"
retain_pair_fixtures "$ROOT_A" "indep-compatible-qa" "indep-compatible-dr"
DIRECT_A="$ROOT_A/direct-comparator-output.json"
"$COMPARATOR" --left "$ROOT_A/scenarios/$SCENARIO_ID/$QA_REP/evaluation.jsonl" \
	--right "$ROOT_A/scenarios/$SCENARIO_ID/$DR_REP/evaluation.jsonl" \
	--mode independent_frozen_candidate --output "$DIRECT_A" \
	|| fail "case A: direct comparator invocation itself failed (fixture is not actually compatible)"

RC_A="$(run_case "case-a" "$ROOT_A" "independent_frozen_candidate")"
[[ "$RC_A" -eq 0 ]] || fail "case A: expected exit 0 (accepted+published), got $RC_A: $(cat "$ROOT_A.out")"
[[ -f "$ROOT_A/scenarios/$SCENARIO_ID/$DR_REP/comparison.json" ]] || fail "case A: accepted comparison was not published"
diff -u "$DIRECT_A" "$ROOT_A/scenarios/$SCENARIO_ID/$DR_REP/comparison.json" >/dev/null \
	|| fail "case A: published comparison.json is not byte-identical to the direct comparator's own output: $(diff -u "$DIRECT_A" "$ROOT_A/scenarios/$SCENARIO_ID/$DR_REP/comparison.json")"
grep -q '"accepted":true' "$ROOT_A/scenarios/$SCENARIO_ID/$DR_REP/comparison.json" \
	|| fail "case A: published comparison.json does not report accepted:true"

echo "TC-087(case A, AC-010): identity-compatible independent pair accepted and published, byte-identical to the real comparator's own output -- PASS"

# ===========================================================================
# Case B: one-field-divergent pair, independent_frozen_candidate -> REJECT,
# NEVER PUBLISHED, divergence reason preserved verbatim.
# ===========================================================================
ROOT_B="$WORKDIR/root-b"
mkdir -p "$ROOT_B"
retain_pair_fixtures "$ROOT_B" "indep-divergent-qa" "indep-divergent-dr"
DIRECT_B="$ROOT_B/direct-comparator-output.json"
set +e
"$COMPARATOR" --left "$ROOT_B/scenarios/$SCENARIO_ID/$QA_REP/evaluation.jsonl" \
	--right "$ROOT_B/scenarios/$SCENARIO_ID/$DR_REP/evaluation.jsonl" \
	--mode independent_frozen_candidate --output "$DIRECT_B"
direct_b_rc=$?
set -e
[[ "$direct_b_rc" -ne 0 ]] || fail "case B: direct comparator invocation unexpectedly accepted the divergent fixture"

RC_B="$(run_case "case-b" "$ROOT_B" "independent_frozen_candidate")"
[[ "$RC_B" -eq 4 ]] || fail "case B: expected exit 4 (rejected, not published), got $RC_B: $(cat "$ROOT_B.out")"
[[ ! -f "$ROOT_B/scenarios/$SCENARIO_ID/$DR_REP/comparison.json" ]] || fail "case B: a REJECTED comparison must never be published"
grep -q "identity_mismatch" "$ROOT_B.out" || fail "case B: driver output did not preserve the comparator's identity_mismatch divergence reason: $(cat "$ROOT_B.out")"

# Byte-for-byte verbatim cross-check: the driver's stdout (the published
# JSON only, diagnostic banners are on stderr, checked separately above)
# must reproduce the exact bytes of the direct comparator's own --output
# file.
diff -u "$DIRECT_B" "$ROOT_B.stdout" >/dev/null \
	|| fail "case B: driver's printed divergence output is not byte-identical to the direct comparator's own output: $(diff -u "$DIRECT_B" "$ROOT_B.stdout")"

echo "TC-087(case B, AC-010/AC-T2): one-field-divergent independent pair rejected, never published, divergence reason preserved verbatim -- PASS"

# ===========================================================================
# Case C: branch-name/HEAD-only candidate identity -> REJECTED by the
# DELEGATED comparator's own malformed-digest check (never an F10 special
# case for the literal string "HEAD" or a branch name).
# ===========================================================================
ROOT_C="$WORKDIR/root-c"
mkdir -p "$ROOT_C"
retain_pair_fixtures "$ROOT_C" "indep-branch-qa" "indep-branch-dr"
RC_C="$(run_case "case-c" "$ROOT_C" "independent_frozen_candidate")"
[[ "$RC_C" -eq 4 ]] || fail "case C: expected exit 4 (branch/HEAD candidate identity rejected), got $RC_C: $(cat "$ROOT_C.out")"
[[ ! -f "$ROOT_C/scenarios/$SCENARIO_ID/$DR_REP/comparison.json" ]] || fail "case C: a branch/HEAD-identity comparison must never be published"
grep -q "malformed_digest" "$ROOT_C.out" || fail "case C: expected the comparator's own malformed_digest rejection for a HEAD-only candidate identity: $(cat "$ROOT_C.out")"

echo "TC-087(case C, REQ-F-015 Negative Case): a HEAD-only candidate identity is rejected by the delegated comparator, not a local F10 special-case -- PASS"

# ===========================================================================
# Case D: workflow-policy-only divergence (candidate identical) -> REJECT
# with a reason distinguishing policy divergence from candidate divergence
# (test-plan.md TC-087 Edge Cases).
# ===========================================================================
ROOT_D="$WORKDIR/root-d"
mkdir -p "$ROOT_D"
retain_pair_fixtures "$ROOT_D" "indep-policy-divergent-qa" "indep-policy-divergent-dr"
RC_D="$(run_case "case-d" "$ROOT_D" "independent_frozen_candidate")"
[[ "$RC_D" -eq 4 ]] || fail "case D: expected exit 4 (policy-only divergence rejected), got $RC_D: $(cat "$ROOT_D.out")"
grep -q '"field": *"workflow_policy/' "$ROOT_D.out" || grep -q '"field":"workflow_policy/' "$ROOT_D.out" \
	|| fail "case D: divergence reason did not name a workflow_policy/* field (must distinguish policy divergence from candidate divergence): $(cat "$ROOT_D.out")"
if grep -q '"field": *"candidate/' "$ROOT_D.out" || grep -q '"field":"candidate/' "$ROOT_D.out"; then
	fail "case D: a candidate/* divergence must not appear when only the workflow policy diverges: $(cat "$ROOT_D.out")"
fi

echo "TC-087(case D, Edge Case): workflow-policy-only divergence rejected with a policy-specific reason, distinct from a candidate divergence -- PASS"

# ===========================================================================
# Case E: three-candidate sequential_delivery chain -> ACCEPT, every
# intervening candidate exercised (not just first-and-last), and the
# comparison output renders a distinct (longer) candidate lineage than the
# independent-mode case A above.
# ===========================================================================
ROOT_E="$WORKDIR/root-e"
mkdir -p "$ROOT_E"
retain_pair_fixtures "$ROOT_E" "seq-chain-qa" "seq-chain-dr"
RC_E="$(run_case "case-e" "$ROOT_E" "sequential_delivery")"
[[ "$RC_E" -eq 0 ]] || fail "case E: expected exit 0 (accepted+published sequential chain), got $RC_E: $(cat "$ROOT_E.out")"
[[ -f "$ROOT_E/scenarios/$SCENARIO_ID/$DR_REP/comparison.json" ]] || fail "case E: accepted sequential comparison was not published"

INTERVENING_COUNT="$(python3 -c 'import json,sys; print(len(json.load(open(sys.argv[1]))["comparison"]["intervening_candidates"]))' "$ROOT_E/scenarios/$SCENARIO_ID/$DR_REP/comparison.json")"
[[ "$INTERVENING_COUNT" -eq 3 ]] || fail "case E: expected all 3 intervening candidates to be exercised (not just first-and-last), got $INTERVENING_COUNT"

INDEPENDENT_INTERVENING_COUNT="$(python3 -c 'import json,sys; print(len(json.load(open(sys.argv[1]))["comparison"]["intervening_candidates"]))' "$ROOT_A/scenarios/$SCENARIO_ID/$DR_REP/comparison.json")"
[[ "$INDEPENDENT_INTERVENING_COUNT" -eq 0 ]] || fail "case A/E cross-check: independent_frozen_candidate mode must render an empty candidate lineage, distinct from sequential_delivery's chain"
[[ "$INTERVENING_COUNT" -gt "$INDEPENDENT_INTERVENING_COUNT" ]] || fail "case E: sequential_delivery must render a distinct (non-empty) candidate lineage vs. independent_frozen_candidate's frozen pair"

echo "TC-087(case E, AC-010): three-candidate sequential_delivery chain accepted and published; every intervening candidate exercised; lineage rendering is distinct from independent_frozen_candidate mode -- PASS"

# ===========================================================================
# Case F (code-review-2026-08-21T0330-E40-F10.md finding 2, swept from
# lib/retain_pair into every write call this driver owns): the ACCEPTED
# comparison publish path (`cp -- ... comparison_dest`, now a temp-file +
# `mv --` atomic install) must not write through a pre-existing symlink
# planted at comparison_dest -- the same defect class finding 2 diagnosed
# in lib/retain_pair, at a different write call in a different file.
#
# The symlink target is deliberately IN-ROOT (the qa gate's own real,
# already-retained evaluation.jsonl), not outside the retention root: a
# symlink pointing OUTSIDE the root at this exact call site is already
# caught earlier by this same publish block's own assert_within_out_root
# check (realpath -m resolves an existing symlink's final component, so an
# outside target already fails containment before any write is attempted
# -- unlike lib/retain_pair's internal per-artifact paths, which no caller
# ever runs assert_within_out_root against at all). The write-through this
# case proves is the narrower, still-real "confused deputy" variant: a
# symlink redirected to ANOTHER retained artifact inside the same root,
# which containment-by-string-path alone cannot distinguish from a
# legitimate destination.
# ===========================================================================
ROOT_F="$WORKDIR/root-f"
mkdir -p "$ROOT_F"
retain_pair_fixtures "$ROOT_F" "indep-compatible-qa" "indep-compatible-dr"

QA_EVAL_F="$ROOT_F/scenarios/$SCENARIO_ID/$QA_REP/evaluation.jsonl"
QA_EVAL_F_ORIGINAL="$(cat "$QA_EVAL_F")"
ln -sf "$QA_EVAL_F" "$ROOT_F/scenarios/$SCENARIO_ID/$DR_REP/comparison.json"

RC_F="$(run_case "case-f" "$ROOT_F" "independent_frozen_candidate")"
[[ "$RC_F" -eq 0 ]] || fail "case F: expected exit 0 (accepted+published) despite a pre-existing symlink at comparison_dest, got $RC_F: $(cat "$ROOT_F.out")"

[[ "$(cat "$QA_EVAL_F")" == "$QA_EVAL_F_ORIGINAL" ]] \
	|| fail "case F: the qa gate's own evaluation.jsonl (symlink target) was overwritten through comparison_dest's pre-existing symlink -- $(cat "$QA_EVAL_F")"
[[ -L "$ROOT_F/scenarios/$SCENARIO_ID/$DR_REP/comparison.json" ]] && fail "case F: comparison.json is still a symlink after publish -- expected it replaced with real content"
grep -q '"accepted":true' "$ROOT_F/scenarios/$SCENARIO_ID/$DR_REP/comparison.json" \
	|| fail "case F: published comparison.json does not report accepted:true (real content was not installed in place of the symlink)"

echo "TC-087(case F, finding 2): the accepted-comparison publish path replaces a pre-existing destination symlink instead of writing through it -- an unrelated in-root retained artifact untouched"

# ---------------------------------------------------------------------------
# AC-T3 (T-E40-F10-014's future static-grep target, proven here too):
# run-review-comparison.sh contains none of the comparator's own
# candidate/policy digest field names -- delegates identity adjudication
# entirely, never re-implements it.
# ---------------------------------------------------------------------------
for forbidden_field in base_commit tree_digest binary_diff_digest changed_path_digest \
	dirty_untracked_manifest test_suite_digest identity_digest prompt_digest \
	rendered_prompt_digest review_bundle_digest deep_review_bundle_digest \
	workflow_policy_identity_digest; do
	if grep -q -- "$forbidden_field" "$COMPARISON"; then
		fail "AC-T3: run-review-comparison.sh contains the comparator's own identity field name '$forbidden_field'"
	fi
done

echo "TC-087(AC-T3): run-review-comparison.sh contains no comparator identity field-name vocabulary -- PASS"

echo "TC-087: pass (identity-compatible pair published byte-identical to the real comparator; divergent/branch-identity/policy-only pairs rejected and never published with verbatim reasons; sequential chain exercises every intervening candidate with a lineage distinct from independent mode)"
