"""Prepare real writer receipts and single-field variations for Go admission tests."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

root, scripts = map(Path, sys.argv[1:3])
base, plan_digest = sys.argv[3:]
directory = root / ".shark" / "byte-receipts"
directory.mkdir()


def git(*args):
    return subprocess.check_output(["git", "-C", root, *args])


def sha(data):
    return hashlib.sha256(data).hexdigest()


def reference(kind, path):
    return dict(kind="bug_review_" + kind, pointer=str(path.relative_to(root)),
                summary="sha256:" + sha(path.read_bytes()))


def save(path, value):
    path.write_text(json.dumps(value, indent=2) + "\n", encoding="utf-8")


byte_name = "caf\udce9.txt"
names = [byte_name, "café.txt", "caf�.txt", "caf\\udce9.txt", "𐃩.txt", " ", 'quoted"name']
for index, name in enumerate(names):
    (root / name).write_text(f"value {index}\n")
git("add", "--", *names)
git("commit", "-qm", "review byte-distinct paths")
expected = [path.decode("utf-8", "surrogateescape")
            for path in git("diff", "--name-only", "-z", base, "HEAD").split(b"\0") if path]
scope_path = directory / "capture.json"
scope_path.write_bytes(subprocess.check_output(["bash", scripts / "get_diff.sh"], cwd=root,
    env={**os.environ, "REVIEW_BASE": base, "REVIEW_PLAN_SHA256": plan_digest}))
scope = json.loads(scope_path.read_bytes())
assert expected == scope["changed_files"] and all(name in expected for name in names)
identity = {
    "commit": scope["candidate_commit"],
    "tree": scope["candidate_tree"],
    "diff_sha256": scope["diff_sha256"],
    "repair_plan_sha256": scope["repair_plan_sha256"],
}
supplemental = "context\udcea.txt"
inspected = []
for index, name in enumerate(names):
    assert (root / name).read_bytes() == f"value {index}\n".encode()
    inspected.append(name)
assert (root / supplemental).read_bytes() == b"unchanged reviewer context\n"
inspected.append(supplemental)
packet = {"review_contract_version": 2, "candidate_identity": identity, "reviewed_files": inspected, "findings": []}
angles = {angle: packet for angle in "abcdef"}
body = directory / "body.md"
body.write_text("Scope: inspected every changed file in the frozen candidate and its callers. Findings: 0 defects found. Verdict: PASS\n")
decision_input, angle_input = directory / "decisions.json", directory / "angles.json"
save(decision_input, packet)
save(angle_input, angles)


def variants(mode):
    report = directory / (mode + ".md")
    cli = mode == "adversarial-cli"
    options = ["--adversarial-model", "codex"] if cli else ["--angle-results", str(angle_input)]
    subprocess.run([sys.executable, scripts / "adaptive_review.py", "write-report",
        "--project-root", root, "--source-root", root, "--scope-file", scope_path, "--review-output-path", report,
        "--runner-mode", mode, "--base-commit", base, "--diff-path", scope["diff_path"],
        "--review-decisions", decision_input, "--body-file", body, "--verdict", "PASS",
        "--specialists-completed", "0" if cli else "6", "--consolidator-completed", "true",
        "--fallback-reason", "preceding route unavailable", *options], check=True, stdout=subprocess.PIPE)
    artifacts = {"report": report}
    for kind in ("scope", "decisions") if cli else ("scope", "decisions", "angles"):
        artifacts[kind] = Path(f"{report}.{kind}.json")

    def trial(case, paths):
        return {"name": f"{mode}-{case}", "refs": [reference(k, p) for k, p in paths.items()]}

    yield trial("complete", artifacts)
    reviewer_field = "decisions" if cli else "angles"
    for field in ("scope", reviewer_field):
        challenges = [("omitted", byte_name, None), ("utf8", byte_name, "café.txt"),
                      ("replacement", byte_name, "caf�.txt")]
        if field == reviewer_field:
            challenges.append(("optional-omitted", supplemental, None))
        for label, original, replacement in challenges:
            sidecar = artifacts[field]
            value = json.loads(sidecar.read_bytes())
            if field == "scope":
                members = value["changed_files"]
            elif field == "angles":
                members = value["b"]["reviewed_files"]
            else:
                members = value["reviewed_files"]
            index = members.index(original)
            if replacement is None:
                members.pop(index)
                if field == "scope":
                    value["changed_file_count"] = len(members)
            else:
                members[index] = replacement
            prefix = directory / f"{mode}-{field}-{label}"
            changed = prefix.with_suffix(".json")
            save(changed, value)
            new_report = prefix.with_suffix(".md")
            content = report.read_text().replace(str(sidecar), str(changed)).replace(
                sha(sidecar.read_bytes()), sha(changed.read_bytes()))
            if field == "scope":
                content = content.replace(json.dumps(expected), json.dumps(members))
            new_report.write_text(content)
            yield trial(f"{field}-{label}", {**artifacts, "report": new_report, field: changed})
    yield trial("restored", artifacts)


try:
    modes = ("canonical-six-angle", "dispatched-six-angle", "adversarial-cli")
    print(json.dumps([case for mode in modes for case in variants(mode)]))
finally:
    Path(scope["diff_path"]).unlink()
