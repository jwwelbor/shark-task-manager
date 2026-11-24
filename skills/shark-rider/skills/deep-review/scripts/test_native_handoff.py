import itertools
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

SCRIPT = Path(__file__).with_name("review_workflow.js")
# Exercise the Workflow body with fixtures only at the host invocation boundary.
HARNESS = r"""
const fs = require('fs');
const src = fs.readFileSync(process.argv[1], 'utf8');
const Workflow = Object.getPrototypeOf(async function() {}).constructor;
const run = new Workflow('args', 'phase', 'parallel', 'agent', 'log', src.slice(src.indexOf('// ─── Args')));
const fixture = JSON.parse(fs.readFileSync(0, 'utf8'));
const calls = [];
const agent = async (prompt, options) => {
    calls.push({prompt, label: options.label});
    if (options.label !== 'consolidator') return fixture.packets[options.label[0].toLowerCase()];
    return {report: 'Scope: checked every file and caller on this candidate. Findings: all six defect observations remain open. Verdict: FAIL',
            decisions: fixture.decisions};
};
const dispatch = async jobs => fixture.noVector ? null : Promise.all(jobs.map(job => job()));
run(fixture.args, () => {}, dispatch, agent, () => {}).then(
    result => process.stdout.write(JSON.stringify({result, calls})),
    error => process.stdout.write(JSON.stringify({error: error.message, calls})));
"""


def exercise(paths, *, encoded=False, no_vector=False, missing=None, manifest=None, extra=None, consolidated=None):
    identity = {"commit": "commit", "tree": "tree", "diff_sha256": "digest"}
    args = {"changed_files": paths if manifest is None else manifest, "diff_path": "diff", "candidate_commit": "commit",
            "candidate_tree": "tree", "diff_sha256": "digest", "skill_dir": str(SCRIPT.parent.parent)}
    packets = {}
    decisions = []
    for index, label in enumerate("abcdef"):
        order = paths[index:] + paths[:index]
        finding = {"file": paths[0], "line": 1, "severity": "blocker", "rule": "CORRECTNESS",
                   "summary": f"observation {label}", "diagnosis": f"missing registration {label}",
                   "evidence": f"observed callback {label}"}
        packets[label] = {"candidate_identity": identity, "findings": [finding],
                          "reviewed_files": [*order, f"context-{label}"]}
        decisions.append(dict(finding, disposition="open", reason="callback is missing"))
    if missing:
        label, member = missing
        packets[label]["reviewed_files"].remove(member)
    if extra is not None:
        packets["a"]["reviewed_files"].append(extra)
    decision_packet = {"candidate_identity": identity, "findings": decisions}
    if consolidated is not None:
        decision_packet["reviewed_files"] = consolidated
    fixture = {"args": json.dumps(args) if encoded else args, "packets": packets, "noVector": no_vector,
               "decisions": decision_packet}
    observed = subprocess.check_output(["node", "-e", HARNESS, str(SCRIPT)],
                                       input=json.dumps(fixture), text=True)
    return json.loads(observed), packets


class NativeHandoffTest(unittest.TestCase):
    def test_each_recipient_gets_git_manifest_and_consolidator_gets_original_packets(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            def git(*args):
                return subprocess.check_output(["git", *args], cwd=root)
            git("init", "-q")
            git("-c", "user.name=Fixture", "-c", "user.email=f@example.invalid",
                "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-qm", "baseline")
            shared = [b" ", b'literal\\file', b'quoted"file', b'caf\xe9', b'caf\xef\xbf\xbd']
            manifests = []
            for group, encoded in [([b"alpha", b"beta\ngamma"], False),
                                   ([b"alpha\nbeta", b"gamma"], True), (shared, True)]:
                git("reset", "--hard", "-q", "HEAD")
                for name in group:
                    (root / os.fsdecode(name)).write_bytes(name)
                git("add", "-A")
                raw = git("diff", "--cached", "--name-only", "-z", "HEAD")
                paths = [name.decode("utf-8", "surrogateescape") for name in raw.split(b"\0")[:-1]]
                manifests.append(paths)
                output, packets = exercise(paths, encoded=encoded)
                calls = output["calls"]
                self.assertEqual(len(calls), 7)
                for call in calls:
                    with self.subTest(group=group, recipient=call["label"]):
                        after = call["prompt"].split("- CHANGED_FILES:", 1)[1].lstrip()
                        self.assertTrue(after.startswith("["), "expected a lossless JSON file list")
                        self.assertEqual(json.JSONDecoder().raw_decode(after)[0], paths)
                with self.subTest(group=group, recipient="consolidator", field="angle packets"):
                    prompt = calls[-1]["prompt"]
                    self.assertIn("- ANGLE_RESULTS:", prompt)
                    after = prompt.split("- ANGLE_RESULTS:", 1)[1].lstrip()
                    self.assertEqual(json.JSONDecoder().raw_decode(after)[0], packets)
                self.assertTrue(output["result"]["consolidator_completed"])
                self.assertEqual(output["result"]["angle_results"], packets)
            self.assertNotEqual(manifests[0], manifests[1])
            self.assertEqual("\n".join(manifests[0]), "\n".join(manifests[1]))

    def test_missing_member_identifies_its_producer_before_consolidation(self):
        paths = [" ", "file\nname", "ordinary"]
        for label, path in itertools.product("abcdef", paths):
            with self.subTest(label=label, path=path):
                observed, _ = exercise(paths, missing=(label, path))
                result = observed["result"]
                self.assertEqual(len(observed["calls"]), 6)
                self.assertEqual(result["specialists_completed"], 5)
                self.assertFalse(result["consolidator_completed"])
                self.assertIn("angle " + label, result["report"].lower())
                self.assertIn(json.dumps(path), result["report"])

    def test_ambiguous_legacy_scope_cannot_satisfy_a_frozen_manifest(self):
        observed, _ = exercise(["a", "b"], manifest="a\nb")
        self.assertEqual(observed["calls"], [])
        self.assertIn("changed_files", observed.get("error", observed.get("result", {}).get("report", "")))
        single, _ = exercise(["one"], manifest="one")
        self.assertTrue(single["result"]["consolidator_completed"])

    def test_each_declaration_preserves_supplemental_path_identity(self):
        paths = [" ", "ordinary"]
        for extra in ["", "../other", "/outside", "nested/../other", "unpaired\ud800", "nul\x00"]:
            with self.subTest(extra=extra):
                observed, _ = exercise(paths, extra=extra)
                self.assertEqual(len(observed["calls"]), 6)
                self.assertEqual(observed["result"]["specialists_completed"], 5)
                self.assertIn("angle a", observed["result"]["report"].lower())
        for declared, valid in [(paths, True), (["ordinary"], False), (paths + ["../other"], False)]:
            with self.subTest(consolidated=declared):
                observed, _ = exercise(paths, consolidated=declared)
                self.assertEqual(len(observed["calls"]), 7)
                self.assertEqual(observed["result"]["specialists_completed"], 6)
                self.assertEqual(observed["result"]["consolidator_completed"], valid)

    def test_dispatch_without_a_result_vector_preserves_unknown_count(self):
        observed, _ = exercise(["file"], no_vector=True)
        self.assertEqual(observed["calls"], [])
        self.assertEqual(observed["result"]["runner_mode"], "incomplete")
        self.assertEqual(observed["result"]["specialists_completed"], "unknown")
        self.assertFalse(observed["result"]["consolidator_completed"])
