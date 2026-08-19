#!/usr/bin/env bash
# TC-075 / X-12 / AC-010: installed workflows, prompts, skills, and agents
# are content identity. Drives the real evaluate-lifecycle.sh content_digest()
# path (bench/scripts/evaluate-lifecycle.sh:100-116, consumed at :269-274) via
# the production CLI entrypoint -- this test must never reimplement digest
# computation and feed it directly to the comparator, because that path never
# exercises evaluate-lifecycle.sh's own mismatch-detection code at all.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
EVALUATOR="$SCRIPT_DIR/../evaluate-lifecycle.sh"
SCENARIO="$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml"
CANONICAL_TREE="$REPO_ROOT/internal/sharkdata/default_data"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/shark-tc075.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT

[[ -x "$EVALUATOR" ]] || { echo "TC-075 FAIL: evaluator missing" >&2; exit 1; }
[[ -f "$SCENARIO" ]] || { echo "TC-075 FAIL: scenario fixture missing" >&2; exit 1; }
[[ -d "$CANONICAL_TREE" ]] || { echo "TC-075 FAIL: canonical Shark-data tree missing" >&2; exit 1; }

python3 - "$TMP_DIR" "$EVALUATOR" "$SCENARIO" "$CANONICAL_TREE" <<'PY'
import copy
import json
import os
import pathlib
import re
import shutil
import subprocess
import sys

root, evaluator, scenario, canonical_tree = (
    pathlib.Path(sys.argv[1]), sys.argv[2], sys.argv[3], pathlib.Path(sys.argv[4]),
)

# Extract the REAL content_digest() source out of evaluate-lifecycle.sh so
# fixture setup below is byte-identical to production -- never a hand-retyped
# reimplementation that can silently drift from it (the defect class this
# test was rejected for). The pass/fail DETERMINATION for every case below is
# always made by invoking the real `evaluate-lifecycle.sh` subprocess, never
# by this helper; the helper only pins the "declared" digest used to set up a
# genuinely matching baseline fixture.
source_text = pathlib.Path(evaluator).read_text(encoding="utf-8")
match = re.search(r"\ndef content_digest\(root\):.*?\n\n\n", source_text, re.S)
assert match, "content_digest() source not found in evaluate-lifecycle.sh"
namespace = {"os": os, "hashlib": __import__("hashlib")}
exec(compile(match.group(0), "evaluate-lifecycle.sh::content_digest", "exec"), namespace)  # noqa: S102
reference_content_digest = namespace["content_digest"]


def identity(content_digest_value):
    return {
        "run_id": "run-tc075", "scenario_id": "py-bug-due-date-boundary", "scenario_version": "1",
        "fixture_id": "f", "fixture_digest": "1" * 64, "adapter_id": "a", "adapter_version": "1",
        "toolchain_identity": [{"key": "tool", "value": "v"}], "shark_binary_digest": "2" * 64,
        "shark_content_digest": content_digest_value, "rendered_prompt_digest": "3" * 64,
        "rendered_prompt_digests": ["3" * 64], "provider": "p", "model": "m", "effort": "e",
        "provider_identity": [{"stage": "code", "provider": "p", "model": "m", "effort": "e"}],
        "judge_digest": "4" * 64, "judge_identity": {"model": "j", "configuration": "c"},
        "reference_digest": "5" * 64, "reference_digests": ["5" * 64], "resource_policy_digest": "6" * 64,
    }


CASE_COUNTER = [0]


def run_evaluator(content_root, content_digest_value):
    CASE_COUNTER[0] += 1
    case_dir = root / f"case-{CASE_COUNTER[0]}"
    i05_dir = case_dir / "i05"
    i05_dir.mkdir(parents=True)
    (i05_dir / "bundle.json").write_text(json.dumps({"content_root": str(content_root)}))
    i07_path = case_dir / "i07.jsonl"
    i07_path.write_text(json.dumps({"identity": identity(content_digest_value)}) + "\n")
    output_path = case_dir / "evaluation.jsonl"
    subprocess.run(
        [evaluator, "--i05", str(i05_dir), "--i07", str(i07_path), "--scenario", scenario, "--output", str(output_path)],
        capture_output=True, text=True, check=False,
    )
    return json.loads(output_path.read_text())


def has_content_mismatch(record):
    return any(
        item["code"] == "identity_mismatch" and item["path"] == "/identity/shark_content_digest"
        for item in record["eligibility"]["invalidity_reasons"]
    )


# --- AC-010 core: real installed canonical tree, one mutation per declared
# class (workflow, prompt, skill, agent). Each case drives evaluate-
# lifecycle.sh's real content_digest() twice: once establishing a genuinely
# matching declared/derived pair (no mismatch), once with the declared value
# stale against a per-class-mutated copy (mismatch, exact source class).
canonical_base = root / "canonical-base"
shutil.copytree(canonical_tree, canonical_base)
base_digest = reference_content_digest(str(canonical_base))
assert base_digest, "reference content_digest() produced no value for the canonical tree"

base_result = run_evaluator(canonical_base, base_digest)
assert not has_content_mismatch(base_result), ("matching base case falsely rejected", base_result)

MUTATION_TARGETS = {
    "workflow": "workflow/task.yaml",
    "prompt": "prompts/task/completed.md",
    "skill": "skills/quality/SKILL.md",
    "agent": "agents/developer.md",
}
for source_class, relative_path in MUTATION_TARGETS.items():
    mutated = root / f"mutated-{source_class}"
    shutil.copytree(canonical_base, mutated)
    target = mutated / relative_path
    assert target.is_file(), (source_class, target)
    target.write_bytes(target.read_bytes() + f"\nTC-075 mutation ({source_class})\n".encode())
    result = run_evaluator(mutated, base_digest)
    assert has_content_mismatch(result), (source_class, result)
    left = {"evaluation_id": "left", "identity": identity(base_digest)}
    right = {"evaluation_id": "right", "identity": identity(reference_content_digest(str(mutated)))}
    assert left["identity"]["shark_content_digest"] != right["identity"]["shark_content_digest"], source_class

print("TC-075 AC-010 core PASS")

# --- X-12 inventory-completeness partitions (test-plan.md TC-075): the
# digest must be proven complete against the whole real tree, not just that
# four representative files affect it.

def write_tree(base, files):
    base.mkdir(parents=True, exist_ok=True)
    for relative, content in files.items():
        path = base / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(content)


# 1) Empty tree: a valid, non-crashing digest that is distinguishable from a
# non-empty tree.
empty_base = root / "inv-empty-base"
empty_base.mkdir(parents=True)
empty_digest = reference_content_digest(str(empty_base))
assert empty_digest, "empty tree must still produce a digest"
empty_result = run_evaluator(empty_base, empty_digest)
assert not has_content_mismatch(empty_result), ("empty tree falsely rejected", empty_result)
nonempty = root / "inv-empty-nonempty"
write_tree(nonempty, {"a.txt": b"content"})
mismatch_vs_empty = run_evaluator(nonempty, empty_digest)
assert has_content_mismatch(mismatch_vs_empty), ("non-empty tree not distinguished from empty", mismatch_vs_empty)

# 2) Missing file: removing a file the declared digest was computed over
# must be caught.
two_file_base = root / "inv-missing-base"
write_tree(two_file_base, {"a.txt": b"a", "b.txt": b"b"})
two_file_digest = reference_content_digest(str(two_file_base))
missing_variant = root / "inv-missing-variant"
shutil.copytree(two_file_base, missing_variant)
(missing_variant / "b.txt").unlink()
missing_result = run_evaluator(missing_variant, two_file_digest)
assert has_content_mismatch(missing_result), ("missing file not detected", missing_result)

# 3) Extra file: an additional file beyond what the declared digest covers
# must be caught.
extra_variant = root / "inv-extra-variant"
shutil.copytree(two_file_base, extra_variant)
(extra_variant / "c.txt").write_bytes(b"c")
extra_result = run_evaluator(extra_variant, two_file_digest)
assert has_content_mismatch(extra_result), ("extra file not detected", extra_result)

# 4) Symlink: a symlinked file's TARGET content is what gets hashed, so
# swapping a real file for a symlink to different content must be caught.
symlink_base = root / "inv-symlink-base"
write_tree(symlink_base, {"d.txt": b"original", "target.txt": b"different"})
symlink_digest = reference_content_digest(str(symlink_base))
symlink_variant = root / "inv-symlink-variant"
shutil.copytree(symlink_base, symlink_variant)
(symlink_variant / "d.txt").unlink()
(symlink_variant / "d.txt").symlink_to("target.txt")
symlink_result = run_evaluator(symlink_variant, symlink_digest)
assert has_content_mismatch(symlink_result), ("symlinked content swap not detected", symlink_result)

# 5) Duplicate logical path: a symlinked directory must not be followed
# (content_digest walks with followlinks=False), so it must not duplicate an
# already-hashed file under a second logical path.
dup_base = root / "inv-dup-base"
write_tree(dup_base, {"subdir/e.txt": b"e"})
dup_digest = reference_content_digest(str(dup_base))
dup_variant = root / "inv-dup-variant"
shutil.copytree(dup_base, dup_variant)
(dup_variant / "subdir_link").symlink_to("subdir", target_is_directory=True)
dup_result = run_evaluator(dup_variant, dup_digest)
assert not has_content_mismatch(dup_result), ("symlinked directory duplicate falsely changed the digest", dup_result)

# 6) Reordered manifest: the digest must be invariant to filesystem creation
# order (content_digest sorts files/directories before hashing).
reorder_base = root / "inv-reorder-base"
reorder_base.mkdir(parents=True)
for name in ("z.txt", "a.txt", "m.txt"):
    (reorder_base / name).write_bytes(name.encode())
reorder_digest = reference_content_digest(str(reorder_base))
reorder_variant = root / "inv-reorder-variant"
reorder_variant.mkdir(parents=True)
for name in ("a.txt", "m.txt", "z.txt"):
    (reorder_variant / name).write_bytes(name.encode())
reorder_result = run_evaluator(reorder_variant, reorder_digest)
assert not has_content_mismatch(reorder_result), ("reordered manifest falsely changed the digest", reorder_result)

# 7) Non-canonical path spelling: on a case-sensitive filesystem, differently
# spelled (differently cased) paths ARE different logical paths and must not
# be silently collapsed as equivalent.
spelling_base = root / "inv-spelling-base"
write_tree(spelling_base, {"Note.md": b"hello"})
spelling_digest = reference_content_digest(str(spelling_base))
spelling_variant = root / "inv-spelling-variant"
write_tree(spelling_variant, {"note.md": b"hello"})
spelling_result = run_evaluator(spelling_variant, spelling_digest)
assert has_content_mismatch(spelling_result), ("non-canonical path spelling silently collapsed", spelling_result)

print("TC-075 X-12 inventory partitions PASS")
print("TC-075 PASS")
PY
