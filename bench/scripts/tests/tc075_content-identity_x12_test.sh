#!/usr/bin/env bash
# TC-075 / X-12: installed workflows, prompts, skills, and agents are identity.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/shark-tc075.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT
mkdir -p "$TMP_DIR/bundle/workflows" "$TMP_DIR/bundle/prompts" "$TMP_DIR/bundle/skills" "$TMP_DIR/bundle/agents"
printf 'workflow\n' > "$TMP_DIR/bundle/workflows/task.yaml"
printf 'prompt\n' > "$TMP_DIR/bundle/prompts/task.md"
printf 'skill\n' > "$TMP_DIR/bundle/skills/review.md"
printf 'agent\n' > "$TMP_DIR/bundle/agents/developer.md"

python3 - "$TMP_DIR" "$SCRIPT_DIR/../compare-lifecycle-evaluations.sh" <<'PY'
import copy, hashlib, json, pathlib, subprocess, sys
root = pathlib.Path(sys.argv[1]); comparator = sys.argv[2]
bundle = root / "bundle"

def content_digest(path):
    material = bytearray()
    for item in sorted((p for p in path.rglob("*") if p.is_file()), key=lambda p: p.relative_to(path).as_posix()):
        material.extend(item.relative_to(path).as_posix().encode()); material.append(0)
        material.extend(item.read_bytes()); material.append(0)
    return hashlib.sha256(material).hexdigest()

def record(digest):
    identity = {"scenario_id": "s", "scenario_version": "1", "fixture_id": "f", "fixture_digest": "1" * 64, "adapter_id": "a", "adapter_version": "1", "toolchain_identity": [{"key": "tool", "value": "v"}], "shark_binary_digest": "2" * 64, "shark_content_digest": digest, "rendered_prompt_digest": "3" * 64, "rendered_prompt_digests": ["3" * 64], "provider": "p", "model": "m", "effort": "e", "provider_identity": [{"stage": "code", "provider": "p", "model": "m", "effort": "e"}], "judge_digest": "4" * 64, "judge_identity": {"model": "j", "configuration": "c"}, "reference_digest": "5" * 64, "reference_digests": ["5" * 64], "resource_policy_digest": "6" * 64}
    candidate = {"base_commit": "a" * 40, "tree_digest": "a" * 64, "binary_diff_digest": "b" * 64, "changed_path_digest": "c" * 64, "dirty_untracked_manifest": "d" * 64, "test_suite_digest": "e" * 64, "scratch_content_digest": "f" * 64, "identity_digest": "1" * 64}
    policy = {"enabled_gates": ["qa"], "gate_order": ["qa"], "reviewer": {"provider": "p", "model": "m", "effort": "e"}, "prompt_digest": "7" * 64, "rendered_prompt_digest": "7" * 64, "review_bundle_digest": "8" * 64, "deep_review_bundle_digest": "8" * 64, "fixes_allowed_between_gates": False, "fix_policy": "none"}
    return {"evaluation_id": "e", "identity": identity, "candidate_snapshots": [{"candidate": candidate}], "workflow_policy": policy, "eligibility": {"aggregate_eligible": True}}

base = record(content_digest(bundle)); changed = copy.deepcopy(base)
for kind in ["workflows", "prompts", "skills", "agents"]:
    item = next((bundle / kind).iterdir()); item.write_text(item.read_text() + "changed\n")
    changed["identity"]["shark_content_digest"] = content_digest(bundle)
    left = root / "left.jsonl"; right = root / "right.jsonl"; out = root / "out.json"
    left.write_text(json.dumps(base) + "\n"); right.write_text(json.dumps(changed) + "\n")
    subprocess.run([comparator, "--left", str(left), "--right", str(right), "--mode", "independent_frozen_candidate", "--output", str(out)], check=False)
    result = json.loads(out.read_text())
    assert not result["accepted"] and any(item["field"] == "identity/shark_content_digest" for item in result["divergences"]), (kind, result)
print("TC-075 PASS")
PY
