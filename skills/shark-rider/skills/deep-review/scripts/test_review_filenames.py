"""Public review writers must preserve Git filename identity through coverage checks."""
import contextlib
import hashlib
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

from test_adaptive_review import HERE, adaptive_review, review_packet, scope_fixture


@unittest.skipUnless(os.name == "posix", "requires byte filenames and executable reviewers")
class ReviewFilenameTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        self.repo, _, self.scope_file = scope_fixture(self.root)
        self.prompt = self.root / "prompt"
        self.prompt.write_text("Inspect the frozen changes.")
        self.body = self.root / "body"
        self.decisions = self.root / "decisions"
        self.run_number = 0

    def git(self, *args):
        return subprocess.check_output(["git", *args], cwd=self.repo)

    def commit_names(self, names):
        base = self.git("rev-parse", "HEAD").decode().strip()
        for name in names:
            with open(os.path.join(os.fsencode(self.repo), name), "ab") as f:
                f.write(name)
        self.git("add", "-A")
        self.git("commit", "-qm", "update fixture")
        capture = subprocess.check_output(["bash", str(HERE / "get_diff.sh")], cwd=self.repo,
                                          env={**os.environ, "REVIEW_BASE": base})
        self.scope_file.write_bytes(capture)
        self.scope = json.loads(capture)
        expected = self.git("diff", "--name-only", "-z", base, "HEAD").split(b"\0")[:-1]
        self.assertEqual([name.encode("utf-8", "surrogateescape")
                          for name in self.scope["changed_files"]], expected)

    def reviewer(self, name, declaration):
        # Read the actual named inputs, including context, before emitting the declaration.
        # Invalid entries in negative controls change the declaration alone.
        packet = review_packet(self.scope)
        packet["decisions"]["reviewed_files"] = declaration
        files = {p.encode("utf-8", "surrogateescape") for p in self.scope["changed_files"]}
        files.add(b" ")
        script = self.root / name
        script.write_text(
            "#!/usr/bin/env python3\n"
            f"for name in {sorted(files)!r}:\n"
            "    with open(name, 'rb') as f: assert f.read().endswith(name)\n"
            f"print({json.dumps(packet)!r})\n")
        script.chmod(0o700)
        return str(script)

    def direct_arguments(self, mode, names):
        packet = review_packet(self.scope)
        six_angles = mode != "adversarial-cli"
        self.body.write_text(packet["report"])
        args = ["write-report", "--body-file", str(self.body), "--review-decisions",
                str(self.decisions), "--runner-mode", mode, "--adversarial-model", "codex",
                "--fallback-reason", "dispatch unavailable", "--specialists-completed",
                "6" if six_angles else "0", "--consolidator-completed", "true", "--verdict", "PASS"]
        if six_angles:
            angles = {a: packet["decisions"].copy() for a in "abcdef"}
            angles["b"]["reviewed_files"] = names
            angle_file = self.root / "angles.json"
            angle_file.write_text(json.dumps(angles))
            args.extend(["--angle-results", str(angle_file)])
            del packet["decisions"]["reviewed_files"]
        else:
            packet["decisions"]["reviewed_files"] = names
        self.decisions.write_text(json.dumps(packet["decisions"]))
        return args

    def cli_arguments(self, names, fallback):
        preferred = [] if fallback else names
        return ["run-cli", "--prompt-file", str(self.prompt), "--host", "claude",
                "--no-workflow-available", "--no-agent-dispatch-available",
                "--codex-command", self.reviewer("codex", preferred),
                "--claude-command", self.reviewer("claude", names)]

    def check_writer(self, entry, names, error=None, fallback=False):
        self.run_number += 1
        report = self.root / f"report-{self.run_number}.md"
        common = ["--scope-file", str(self.scope_file), "--base-commit", self.scope["base_commit"],
                  "--diff-path", self.scope["diff_path"], "--project-root", str(self.repo),
                  "--source-root", str(self.repo), "--review-output-path", str(report)]
        if entry in ("run-cli", "final-binding"):
            args = self.cli_arguments(names, fallback)
        else:
            mode = "adversarial-cli" if entry == "write-report" else entry
            args = self.direct_arguments(mode, names)
        args.extend(common)
        if entry == "final-binding":
            selected = review_packet(self.scope)
            selected["decisions"]["reviewed_files"] = names
            with patch.object(adaptive_review, "_try_cli_candidates",
                              return_value=(selected["report"], selected["decisions"], "codex", [])), \
                    contextlib.redirect_stdout(io.StringIO()) as output:
                status = adaptive_review.run_cli(adaptive_review.parser().parse_args(args))
            stdout = output.getvalue()
        else:
            result = subprocess.run([sys.executable, str(HERE / "adaptive_review.py"), *args],
                                    capture_output=True)
            status, stdout = result.returncode, result.stdout
            self.assertEqual(status, 2 if error else 0, (stdout, result.stderr))

        self.assertEqual(status, 2 if error else 0, stdout)
        metadata = json.loads(stdout)
        saved = report.read_text(encoding="utf-8")
        six = entry in ("canonical-six-angle", "dispatched-six-angle")
        count = 6 - int(error is not None) if six else 0
        self.assertEqual(metadata["verdict"], "INCOMPLETE" if error else "PASS")
        self.assertEqual(metadata["specialists_completed"], count)
        self.assertIs(metadata["consolidator_completed"], error is None)
        self.assertIn(f"**Verdict:** {metadata['verdict']}", saved)
        self.assertIn(f"`specialists_completed={count}`", saved)
        self.assertIn(f"`consolidator_completed={error is None}`", saved)
        if error:
            self.assertIn(error, metadata["fallback_reason"])
            self.assertIn(error, saved)
            if six:
                self.assertIn("angle b", metadata["fallback_reason"])
            if entry == "final-binding":
                self.assertIn("no longer match the validated scope", metadata["fallback_reason"])
        elif entry == "run-cli":
            self.assertEqual(metadata["adversarial_model"], "claude" if fallback else "codex")

        if six:
            self.assert_declaration(metadata, saved, "angles", names, angle="b")
        elif not error or entry == "write-report":
            self.assert_declaration(metadata, saved, "decisions", names)

    def assert_declaration(self, metadata, saved, kind, names, angle=None):
        data = Path(metadata["retained_" + kind]).read_bytes()
        packet = json.loads(data)
        if angle:
            packet = packet[angle]
        self.assertEqual(packet["reviewed_files"], names)
        digest = hashlib.sha256(data).hexdigest()
        self.assertEqual(digest, metadata[kind + "_sha256"])
        self.assertIn(digest, saved)

    def test_scope_and_snapshot_use_the_same_git_names_across_filesystem_encodings(self):
        names = [b"caf\xe9", "café".encode(), "caf\ufffd".encode(), br"caf\udce9", "caf\U000100e9".encode()]
        self.commit_names(names)
        base = self.scope["candidate_commit"]
        for name in names:
            with open(os.fsencode(self.repo) + b"/" + name, "ab") as f:
                f.write(b"selected")
        envs = [{**os.environ, "LC_ALL": "C", "PYTHONCOERCECLOCALE": "0", "PYTHONUTF8": mode}
                for mode in ("0", "1")]
        observed_encodings = [subprocess.check_output(
            [sys.executable, "-c", "import sys; print(sys.getfilesystemencoding())"], env=env, text=True
        ).strip() for env in envs]
        self.assertEqual(observed_encodings, ["ascii", "utf-8"])
        expected = [name.decode("utf-8", "surrogateescape") for name in names]
        for index, producer in enumerate(envs):
            snapshot = self.root / f"snapshot-{index}"
            metadata = self.root / f"source-{index}.json"
            subprocess.run([sys.executable, str(HERE / "prepare_candidate.py"), "--project-root", self.repo,
                            "--destination", snapshot, "--metadata-file", metadata, *names],
                           env=producer, check=True, capture_output=True)
            marker = subprocess.check_output(["git", "rev-parse", "--git-path", "shark-review-source.json"],
                                             cwd=snapshot, text=True).strip()
            with self.subTest(producer=observed_encodings[index]):
                self.assertEqual(json.loads(metadata.read_bytes())["selected_paths"], expected)
                self.assertEqual(json.loads((snapshot / marker).read_bytes())["selected_paths"], expected)
            changed = subprocess.check_output(["git", "diff", "--name-only", "-z", base, "HEAD"],
                                              cwd=snapshot).split(b"\0")[:-1]
            for name in names:
                with open(os.fsencode(snapshot) + b"/" + name, "rb") as f:
                    self.assertEqual(f.read(), name + b"selected")
            for host, consumer in zip(observed_encodings, envs):
                capture = subprocess.check_output(["bash", str(HERE / "get_diff.sh")], cwd=snapshot,
                          env={**consumer, "REVIEW_BASE": base, "REVIEW_SOURCE_METADATA": str(metadata)})
                self.scope_file.write_bytes(capture)
                with self.subTest(producer=index, capture=host):
                    self.assertEqual(json.loads(capture)["changed_files"],
                                     [name.decode("utf-8", "surrogateescape") for name in changed])
                for reader, env in zip(observed_encodings, envs):
                    with self.subTest(producer=index, capture=host, reader=reader):
                        verified = subprocess.run([sys.executable, str(HERE / "adaptive_review.py"),
                            "validate-scope", "--scope-file", self.scope_file,
                            "--project-root", snapshot, "--source-root", self.repo],
                            env=env, capture_output=True, text=True)
                        self.assertEqual(verified.returncode, 0, (verified.stdout, verified.stderr))
                        self.assertTrue(json.loads(verified.stdout)["valid"])

    def colliding_manifests(self):
        # Both pairs flatten to the same lines, and all decoy paths exist in both trees.
        common = [br'literal\file', b'quoted"file']
        pairs = [[b'alpha', b'beta\ngamma'], [b'alpha\nbeta', b'gamma']]
        self.commit_names(common + pairs[0] + pairs[1])
        base = self.scope["candidate_commit"]
        for pair in pairs:
            self.git("checkout", "-q", "--detach", base)
            self.commit_names(pair + common)
            expected = self.git("diff", "--name-only", "-z", base, "HEAD").split(b"\0")[:-1]
            yield [name.decode() for name in expected]

    def test_native_recipients_receive_exact_git_manifest(self):
        harness = r"""
const fs = require('fs');
const source = fs.readFileSync(process.argv[1], 'utf8');
const run = new (Object.getPrototypeOf(async function() {}).constructor)(
    'args', 'phase', 'parallel', 'agent', 'log', source.slice(source.indexOf('// ─── Args')));
const {scope, packet} = JSON.parse(fs.readFileSync(0, 'utf8'));
const calls = [];
async function reviewer(prompt, {label}) {
    const field = prompt.split('- CHANGED_FILES:\n')[1].split('\n- ')[0];
    const observation = {label, field};
    calls.push(observation);
    try {
        observation.contents = JSON.parse(field).map(name => fs.readFileSync(name, 'utf8'));
    } catch (error) {
        observation.readError = error.message;
    }
    return label === 'consolidator' ? packet : packet.decisions;
}
run(scope, () => {}, tasks => Promise.all(tasks.map(task => task())), reviewer, () => {})
    .then(result => process.stdout.write(JSON.stringify({result, calls})));
"""
        for expected in self.colliding_manifests():
            trial = {"scope": {**self.scope, "skill_dir": str(HERE.parent)},
                     "packet": review_packet(self.scope)}
            output = subprocess.check_output(["node", "-e", harness, str(HERE / "review_workflow.js")],
                                             input=json.dumps(trial), text=True, cwd=self.repo)
            observed = json.loads(output)
            labels = [call["label"].split(":")[0] for call in observed["calls"]]
            self.assertEqual(labels, [*"ABCDEF", "consolidator"])
            self.assertTrue(observed["result"]["consolidator_completed"])
            for call in observed["calls"]:
                with self.subTest(candidate=self.scope["candidate_commit"], recipient=call["label"]):
                    self.assertNotIn("readError", call, call["field"])
                    self.assertEqual(json.loads(call["field"]), expected)
                    self.assertEqual(call["contents"], [name * 2 for name in expected])

            trial["scope"]["changed_files"] = "\n".join(expected)
            output = subprocess.check_output(["node", "-e", harness, str(HERE / "review_workflow.js")],
                                             input=json.dumps(trial), text=True, cwd=self.repo)
            rejected = json.loads(output)
            with self.subTest(invalid_input="newline-delimited manifest"):
                self.assertEqual(rejected["calls"], [])
                self.assertEqual(rejected["result"]["runner_mode"], "incomplete")
                self.assertIn("changed_files", rejected["result"]["report"])

    def test_cli_manifest_at_preferred_and_fallback_executables(self):
        for expected in self.colliding_manifests():
            for fallback in (False, True):
                with self.subTest(candidate=self.scope["candidate_commit"], fallback=fallback):
                    packet = review_packet(self.scope)
                    for cli in ("codex", "claude"):
                        executable = self.root / cli
                        executable.write_text(
                            "#!/usr/bin/env python3\n"
                            "import json, pathlib, sys\n"
                            "prompt = sys.argv[1]\n"
                            "marker = '- CHANGED_FILES:\\n'\n"
                            "field = prompt.split(marker, 1)[1].split('\\n', 1)[0] if marker in prompt else None\n"
                            "paths = json.loads(field) if field else []\n"
                            "contents = [pathlib.Path(name).read_text() for name in paths]\n"
                            f"with open({str(self.root / 'calls')!r}, 'a') as log:\n"
                            f"    print(json.dumps([{cli!r}, field, contents]), file=log)\n"
                            f"print({json.dumps(packet)!r})\n"
                            f"sys.exit({int(fallback and cli == 'codex')})\n")
                        executable.chmod(0o700)
                    (self.root / "calls").write_text("")
                    report = self.root / f"report-{self.scope['candidate_commit']}-{fallback}.md"
                    args = ["run-cli", "--prompt-file", str(self.prompt), "--host", "claude",
                            "--no-workflow-available", "--no-agent-dispatch-available",
                            "--codex-command", str(self.root / "codex"), "--claude-command", str(self.root / "claude"),
                            "--scope-file", str(self.scope_file), "--base-commit", self.scope["base_commit"],
                            "--diff-path", self.scope["diff_path"], "--project-root", str(self.repo),
                            "--source-root", str(self.repo), "--review-output-path", str(report)]
                    output = subprocess.check_output([sys.executable, str(HERE / "adaptive_review.py"), *args])
                    metadata = json.loads(output)
                    calls = [json.loads(line) for line in (self.root / "calls").read_text().splitlines()]
                    self.assertEqual([c[0] for c in calls], ["codex", "claude"] if fallback else ["codex"])
                    self.assertEqual(metadata["adversarial_model"], calls[-1][0])
                    self.assertEqual(metadata["verdict"], "PASS")
                    for cli, field, contents in calls:
                        self.assertIsNotNone(field, cli)
                        self.assertEqual(json.loads(field), expected, cli)
                        self.assertEqual(contents, [name * 2 for name in expected], cli)

    def test_required_and_supplemental_space_across_cli_writers(self):
        self.commit_names([b"ordinary", b" "])
        base = self.git("rev-parse", "HEAD").decode().strip()
        for required in (True, False):
            self.git("checkout", "-q", "--detach", base)
            self.commit_names([b"ordinary", b" "] if required else [b"ordinary"])
            for entry in ("run-cli", "final-binding", "write-report"):
                with self.subTest(entry=entry, required=required):
                    self.check_writer(entry, ["ordinary", " "])
                    self.check_writer(entry, ["ordinary"], '" "' if required else None)
                    for bad in ("", "../outside"):
                        self.check_writer(entry, ["ordinary", bad], json.dumps(bad))
                    self.check_writer(entry, ["ordinary", " "], fallback=entry == "run-cli")

    def test_distinct_byte_names_in_omission_diagnostic(self):
        raw = "caf\udce9"
        names = [b"caf\xe9", "café".encode(), "caf\ufffd".encode(),
                 br"caf\udce9", "caf\U000100e9".encode()]
        self.commit_names([b" ", names[0]])
        self.commit_names(names[1:])
        self.commit_names(names)
        full = self.scope["changed_files"]
        for entry in ("canonical-six-angle", "dispatched-six-angle", "run-cli", "final-binding", "write-report"):
            for replacement in ([], ["café"], ["caf\ufffd"]):
                with self.subTest(entry=entry, replacement=replacement):
                    self.check_writer(entry, [p for p in full if p != raw] + replacement,
                                      json.dumps(raw))
            self.check_writer(entry, full)

    def test_angle_declarations_validate_supplemental_paths(self):
        self.commit_names([b" ", b"ordinary"])
        for entry in ("canonical-six-angle", "dispatched-six-angle"):
            for bad in ("", "sub/./path", "sub//path", "../outside", "/absolute", "a\0b", "\ud800"):
                with self.subTest(entry=entry, bad=bad):
                    self.check_writer(entry, ["ordinary", " ", bad], json.dumps(bad))
            self.check_writer(entry, ["ordinary", " ", "source"])
