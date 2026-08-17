#!/usr/bin/env bash
# TC-072: independent frozen and sequential delivery are distinct contracts.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPARATOR="$SCRIPT_DIR/../compare-lifecycle-evaluations.sh"
WORKDIR="$(mktemp -d "${TMPDIR:-/tmp}/shark-tc072.XXXXXX")"
trap 'rm -rf "$WORKDIR"' EXIT

python3 - "$WORKDIR" "$COMPARATOR" <<'PY'
import copy, json, pathlib, subprocess, sys
root, comparator = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
digest = "a" * 64
identity = {key: value for key, value in {
    "scenario_id": "s", "scenario_version": "1", "fixture_id": "f", "fixture_digest": digest,
    "adapter_id": "a", "adapter_version": "1", "toolchain_identity": [{"key":"go","value":"1"}],
    "shark_binary_digest": digest, "shark_content_digest": digest, "rendered_prompt_digest": digest,
    "rendered_prompt_digests": [digest], "provider": "fixture", "model": "m", "effort": "low",
    "provider_identity": [{"stage":"qa","provider":"fixture","model":"m","effort":"low"}],
    "judge_digest": digest, "judge_identity": {"model":"j","configuration":"c"},
    "reference_digest": digest, "reference_digests": [digest], "resource_policy_digest": digest,
}.items()}
candidate = {"base_commit":"b"*40, "tree_digest":digest, "binary_diff_digest":digest,
             "changed_path_digest":digest, "dirty_untracked_manifest":digest,
             "test_suite_digest":digest, "scratch_content_digest":digest, "identity_digest":digest}
policy = {"enabled_gates":["qa","deep_review"], "gate_order":["qa","deep_review"],
          "reviewer":{"provider":"fixture","model":"m","effort":"low"},
          "prompt_digest":digest, "rendered_prompt_digest":digest,
          "review_bundle_digest":digest, "deep_review_bundle_digest":digest,
          "fixes_allowed_between_gates":False, "fix_policy":"none"}
finding = lambda fid, gate, confirmed: {"f09_finding_id":fid, "gate":gate,
    "final_disposition":"confirmed" if confirmed else "unconfirmed", "confirmation_source":"seeded_truth_set" if confirmed else "none"}
base = {"identity":identity, "workflow_policy":policy, "eligibility":{"aggregate_eligible":True},
        "review_findings":{"normalized_findings":[finding("f09-1","qa",True)], "derived_counts":{"confirmed":1}}}
independent = copy.deepcopy(base); independent.update({"evaluation_id":"independent", "candidate_snapshots":[{"stage":"qa","candidate":candidate}]})
independent["review_findings"]["normalized_findings"].append(finding("f09-1","deep_review",True))
sequential = copy.deepcopy(base); sequential.update({"evaluation_id":"sequential", "candidate_snapshots":[
    {"stage":"qa","candidate":candidate}, {"stage":"deep_review","candidate":dict(candidate, tree_digest="c"*64)}]})
sequential["review_findings"]["normalized_findings"].append(finding("f09-2","deep_review",True))
qa_only = copy.deepcopy(base); qa_only.update({"evaluation_id":"qa-only", "candidate_snapshots":[{"stage":"qa","candidate":candidate}]})
def write(name, record):
    path = root / f"{name}.jsonl"; path.write_text(json.dumps(record)+"\n"); return path
def run(left, right, mode, expect_accepted=True):
    out = root / f"{mode}.json"; proc = subprocess.run([str(comparator), "--left", str(write("left", left)), "--right", str(write("right", right)), "--mode", mode, "--output", str(out)], capture_output=True, text=True)
    result = json.loads(out.read_text())
    assert result["accepted"] is expect_accepted, (proc.stderr, result)
    assert (proc.returncode == 0) is expect_accepted, (proc.returncode, result)
    return result
result = run(independent, independent, "independent_frozen_candidate")
assert result["accepted"] is True and result["comparison"]["causal_claim"] is None, result
result = run(qa_only, sequential, "sequential_delivery")
assert result["accepted"] is True, result
assert result["comparison"]["newly_confirmed_findings"] == ["f09-2"], result
assert len(result["comparison"]["intervening_candidates"]) == 2, result
bad = copy.deepcopy(sequential); bad["candidate_snapshots"] = [sequential["candidate_snapshots"][0]]
result = run(sequential, bad, "sequential_delivery", expect_accepted=False)
assert result["accepted"] is False and any(d["reason"] == "candidate_lineage_missing" for d in result["divergences"]), result
print("TC-072 PASS")
PY
