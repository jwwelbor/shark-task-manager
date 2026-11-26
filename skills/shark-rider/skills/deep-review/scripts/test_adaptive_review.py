#!/usr/bin/env python3
import argparse
import contextlib
import hashlib
import io
import json
import os
import re
import shlex
import stat
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import patch
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import adaptive_review  # noqa: E402
from adaptive_review import (  # noqa: E402
    Capabilities,
    cli_candidates,
    select_runner,
    valid_adversarial_output,
    validate_review_decisions,
    run_bounded_reviewer,
)


class BoundedReviewerTest(unittest.TestCase):
    def test_windows_fails_closed_before_launch(self):
        with patch('adaptive_review.os.name', 'nt'), patch('adaptive_review.subprocess.Popen') as process:
            with self.assertRaisesRegex(OSError, 'process-tree cleanup'):
                run_bounded_reviewer(['reviewer'], None, 1)
            process.assert_not_called()

    @unittest.skipIf(os.name == 'nt', 'uses POSIX process groups')
    def test_excess_output_is_bounded_and_rejected(self):
        script = "import sys; sys.stdout.write('x' * (9 * 1024 * 1024)); sys.stdout.flush()"
        result = run_bounded_reviewer([sys.executable, '-c', script], None, 5)
        self.assertNotEqual(result.returncode, 0)
        self.assertLessEqual(len(result.stdout), 8 * 1024 * 1024)

    @unittest.skipIf(os.name == 'nt', 'uses POSIX process groups')
    def test_closed_output_pipes_do_not_bypass_timeout(self):
        script = "import os,time; os.close(1); os.close(2); time.sleep(10)"
        start = time.monotonic()
        result = run_bounded_reviewer([sys.executable, '-c', script], None, 0.2)
        self.assertNotEqual(result.returncode, 0)
        self.assertLess(time.monotonic() - start, 3)

    @unittest.skipIf(os.name == 'nt', 'uses POSIX process groups')
    def test_parent_exit_with_child_holding_stderr_is_bounded(self):
        with tempfile.TemporaryDirectory() as tmp:
            marker = Path(tmp) / 'child-survived'
            child = "import time,pathlib; time.sleep(1); pathlib.Path(" + repr(str(marker)) + ").write_text('alive')"
            parent = "import subprocess,sys; subprocess.Popen([sys.executable, '-c', " + repr(child) + "], stdout=subprocess.DEVNULL); print('parent exited')"
            result = run_bounded_reviewer([sys.executable, '-c', parent], tmp, 0.2)
            self.assertNotEqual(result.returncode, 0)
            time.sleep(1.1)
            self.assertFalse(marker.exists())


def scope_fixture(root):
    repo = root / 'repo'
    repo.mkdir()
    def git(*args):
        return subprocess.run(['git', '-c', 'commit.gpgsign=false', '-c', 'core.hooksPath=/dev/null', *args], cwd=repo, check=True, capture_output=True, text=True).stdout.strip()
    git('init', '-b', 'main')
    git('config', 'commit.gpgsign', 'false')
    git('config', 'user.name', 'Test')
    git('config', 'user.email', 'test@example.invalid')
    (repo / 'source').write_text('old')
    git('add', 'source')
    git('commit', '-m', 'base')
    git('checkout', '-b', 'candidate')
    (repo / 'source').write_text('new')
    git('commit', '-am', 'candidate')
    capture = subprocess.run(['bash', str(HERE / 'get_diff.sh')], cwd=repo, check=True, capture_output=True, text=True)
    scope = json.loads(capture.stdout)
    scope_file = root / 'scope.json'
    scope_file.write_text(capture.stdout)
    return repo, scope, scope_file


def review_packet(scope):
    identity = {'commit': scope['candidate_commit'], 'tree': scope['candidate_tree'], 'diff_sha256': scope['diff_sha256']}
    return {'report': 'REVIEWED SCOPE: fixture files and coverage\nFINDINGS: none\nThe adversarial review is complete and evidence was checked.\nVERDICT: PASS',
            'decisions': {'candidate_identity': identity, 'reviewed_files': list(scope['changed_files']), 'findings': []}}


def verdict_packet(scope, verdict):
    identity = {'commit': scope['candidate_commit'], 'tree': scope['candidate_tree'], 'diff_sha256': scope['diff_sha256']}
    if verdict == 'PASS':
        report = ('Scope: every changed file and related caller was inspected. Findings: none. '
                  'The frozen candidate was checked against the complete manifest. VERDICT: PASS')
        finding = None
    elif verdict == 'PASS-WITH-TRIAGE':
        report = ('CURRENT_REVIEW_STATUS: COMPLETE\nScope: every changed file and related caller was inspected. '
                  'Findings: one substantive validation issue is retained. VERDICT: PASS-WITH-TRIAGE\n'
                  'FINDING_DETAIL: INCOMPLETE VALIDATION | F-1 | non-blocker | triage')
        finding = {'finding_id': 'F-1', 'file': 'source', 'line': 1, 'severity': 'non-blocker',
                   'rule': 'INPUT', 'summary': 'validation issue', 'diagnosis': 'required input is unchecked',
                   'disposition': 'triage', 'reason': 'tracked for host triage'}
    else:
        report = ('CURRENT_REVIEW_STATUS: COMPLETE\nScope: every changed file and related caller was inspected. '
                  'Findings: a blocking validation defect remains. VERDICT: FAIL\n'
                  'FINDING_DETAIL: INCOMPLETE VALIDATION | F-1 | blocker | open')
        finding = {'finding_id': 'F-1', 'file': 'source', 'line': 1, 'severity': 'blocker',
                   'rule': 'INPUT', 'summary': 'validation defect', 'diagnosis': 'required input is unchecked',
                   'disposition': 'open', 'reason': 'candidate evidence confirms the defect'}
    decisions = {'candidate_identity': identity, 'reviewed_files': list(scope['changed_files']),
                 'findings': [] if finding is None else [finding]}
    return {'report': report, 'decisions': decisions}


def fake_reviewer(path, packet):
    path.write_text("#!/bin/sh\nprintf '%s\\n' " + shlex.quote(json.dumps(packet)) + "\n")
    path.chmod(path.stat().st_mode | stat.S_IXUSR)


class AdaptiveReviewTests(unittest.TestCase):
    @unittest.skipIf(os.name == 'nt', 'uses an executable reviewer fixture')
    def test_public_writers_preserve_the_requested_and_observed_models(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            packet = review_packet(scope)
            reviewer = root / "codex"
            fake_reviewer(reviewer, packet)
            body, decisions, prompt = (root / name for name in ("body", "decisions", "prompt"))
            body.write_text(packet["report"])
            decisions.write_text(json.dumps(packet["decisions"]))
            prompt.write_text("Review the captured changes.")
            entries = {
                "run-cli": ["--codex-command", str(reviewer), "--host", "claude",
                            "--no-workflow-available", "--no-agent-dispatch-available",
                            "--prompt-file", str(prompt)],
                "write-report": ["--body-file", str(body), "--review-decisions", str(decisions),
                                 "--runner-mode", "adversarial-cli", "--adversarial-model", "codex",
                                 "--fallback-reason", "dispatch unavailable", "--specialists-completed", "0",
                                 "--consolidator-completed", "true", "--verdict", "PASS"],
            }
            for entry, options in entries.items():
                for requested in (None, "", "requested-test"):
                    for observed in (None, "observed-test"):
                        with self.subTest(entry=entry, requested=requested, observed=observed):
                            report = root / f"{entry}-{requested}-{observed}.md"
                            argv = [sys.executable, str(HERE / "adaptive_review.py"), entry, *options,
                                    "--project-root", str(repo), "--source-root", str(repo),
                                    "--base-commit", scope["base_commit"], "--scope-file", str(scope_file),
                                    "--diff-path", scope["diff_path"], "--review-output-path", str(report)]
                            if requested is not None:
                                argv.extend(["--requested-model", requested])
                            if observed is not None:
                                argv.extend(["--effective-model", observed])
                            run = subprocess.run(argv, capture_output=True, text=True)
                            self.assertEqual(run.returncode, 0, (argv, run.stderr, run.stdout))
                            result = json.loads(run.stdout)
                            self.assertEqual(result["verdict"], "PASS")
                            self.assertEqual(result["requested_model"], requested)
                            self.assertEqual(result["effective_model"], observed)
                            request_label = "unknown" if requested is None else requested
                            self.assertIn(f"**Requested model:** `{request_label}` · "
                                          f"**Verified effective model:** `{observed or 'unknown'}`",
                                          report.read_text().split("\n---\n", 1)[0])
    def test_incomplete_writer_counts_the_retained_specialist_replies(self):
        # B087 F3: a failed consolidation does not undo completed inspections.
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            reply = review_packet(scope)['decisions']
            body = root / 'body.md'
            reason = 'consolidator returned a malformed report'
            body.write_text('INCOMPLETE: ' + reason)
            angle_file = root / 'angles.json'
            for usable, claimed in [(6, 0), (6, 'unknown'), (6, 6), (5, 6), (0, 'unknown'), (None, 0)]:
                with self.subTest(usable=usable, claimed=claimed):
                    packets = None
                    expected = 'unknown' if usable is None else usable
                    if usable is not None:
                        packets = {label: reply if i < usable else {} for i, label in enumerate('abcdef')}
                    angle_file.write_text(json.dumps(packets))
                    report = root / f'report-{usable}-{claimed}.md'
                    diagnostic = reason if usable == 6 else 'specialist dispatch incomplete'
                    result = subprocess.run([
                        sys.executable, str(HERE / 'adaptive_review.py'), 'write-report',
                        '--runner-mode', 'incomplete', '--specialists-completed', str(claimed),
                        '--consolidator-completed', 'false', '--verdict', 'INCOMPLETE',
                        '--angle-results', str(angle_file), '--fallback-reason', diagnostic,
                        '--body-file', str(body), '--review-output-path', str(report),
                        '--scope-file', str(scope_file), '--project-root', str(repo),
                        '--source-root', str(repo), '--diff-path', scope['diff_path'],
                        '--base-commit', scope['base_commit'],
                    ], capture_output=True, text=True)
                    self.assertEqual(result.returncode, 2, result.stderr)
                    metadata = json.loads(result.stdout)
                    self.assertEqual(metadata['specialists_completed'], expected)
                    self.assertIs(metadata['consolidator_completed'], False)
                    self.assertEqual(metadata['verdict'], 'INCOMPLETE')
                    self.assertEqual(Path(metadata['retained_angles']).read_bytes(), angle_file.read_bytes())
                    saved = report.read_text()
                    self.assertIn(f'`specialists_completed={expected}`', saved)
                    self.assertIn(diagnostic, metadata['fallback_reason'])
                    if claimed != expected:
                        self.assertIn(f'specialists_completed={claimed}', metadata['fallback_reason'])
                        self.assertIn(metadata['fallback_reason'], saved)

    @unittest.skipIf(os.name == 'nt', 'uses byte filenames and executable fixtures')
    def test_cli_selects_only_utf8_packets_and_attributes_the_selected_attempt(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, _, scope_file = scope_fixture(root)
            prompt = root / 'prompt'
            prompt.write_text('Inspect the frozen changes.')
            attempts = root / 'attempts'
            replacement = b'caf\xef\xbf\xbd.txt'
            raw_byte = b'caf\xe9.txt'
            for filename in (replacement, raw_byte):
                base = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=repo, text=True).strip()
                (repo / os.fsdecode(filename)).write_text('review this change')
                subprocess.run(['git', 'add', '--', filename], cwd=repo, check=True)
                subprocess.run(['git', 'commit', '-qm', 'change named file'], cwd=repo, check=True)
                capture = subprocess.check_output(['bash', str(HERE / 'get_diff.sh')], cwd=repo,
                                                  env={**os.environ, 'REVIEW_BASE': base})
                scope_file.write_bytes(capture)
                scope = json.loads(capture)
                self.assertEqual([os.fsencode(p) for p in scope['changed_files']], [filename])

                def reviewer(cli, spelling, diagnostic):
                    path = root / cli
                    packet = review_packet(scope)
                    packet['report'] += '\nReview supplied by ' + cli
                    packet['decisions']['reviewed_files'] = ['WIRE_NAME']
                    if cli == 'claude':
                        packet['decisions']['reviewed_files'].append('source')
                    wire = json.dumps(packet).encode().replace(b'WIRE_NAME', spelling)
                    path.write_text(
                        '#!/usr/bin/env python3\nimport sys\nfrom pathlib import Path\n'
                        f'with open({filename!r}, "rb") as f: assert f.read() == b"review this change"\n'
                        + ('assert Path("source").read_text() == "new"\n' if cli == 'claude' else '')
                        + f'with open({str(attempts)!r}, "a") as f: print({cli!r}, file=f)\n'
                        + f'sys.stdout.buffer.write({wire!r})\nsys.stderr.buffer.write({diagnostic!r})\n')
                    path.chmod(0o700)
                    return path

                exact = filename if filename == replacement else br'caf\udce9.txt'
                cases = [('literal-or-escaped', exact, raw_byte, 'codex'),
                         ('fallback', raw_byte, exact, 'claude'),
                         ('verified-fallback', raw_byte, exact, 'claude'),
                         ('both-malformed', raw_byte, raw_byte, None),
                         ('restored', exact, raw_byte, 'codex')]
                if filename == replacement:
                    cases.append(('both-usable', exact, exact, 'codex'))
                    cases.append(('different-byte', br'caf\udce9.txt', br'caf\udce9.txt', None))
                # R40 F3: change only the producing attempt's stderr, keeping both
                # stdout packets and the other attempt's diagnostic unchanged.
                trials = [(*case, b'\xe9') for case in cases]
                if filename == replacement:
                    paired = {'literal-or-escaped', 'both-usable', 'verified-fallback'}
                    trials.extend((*case, b'') for case in cases if case[0] in paired)
                for case, preferred, fallback, selected, diagnostic in trials:
                    with self.subTest(filename=filename, case=case, diagnostic=diagnostic):
                        attempts.write_text('')
                        report = root / f'{filename.hex()}-{case}-{diagnostic.hex()}.md'
                        preferred_cli = reviewer('codex', preferred, diagnostic if selected != 'claude' else b'')
                        fallback_cli = reviewer('claude', fallback, diagnostic if selected == 'claude' else b'')
                        result = subprocess.run([sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                            '--project-root', str(repo), '--source-root', str(repo), '--base-commit', base,
                            '--scope-file', str(scope_file), '--diff-path', scope['diff_path'],
                            '--prompt-file', str(prompt), '--review-output-path', str(report),
                            '--no-workflow-available', '--no-agent-dispatch-available', '--host', 'claude',
                            '--requested-model', 'requested', '--effective-model', 'verified-preferred',
                            *(['--fallback-effective-model', 'verified-second'] if case == 'verified-fallback' else []),
                            '--codex-command', str(preferred_cli), '--claude-command', str(fallback_cli)],
                            capture_output=True, text=True)
                        self.assertEqual(result.returncode, 0 if selected else 2, result.stderr)
                        metadata = json.loads(result.stdout)
                        saved = report.read_text()
                        self.assertEqual(attempts.read_text().splitlines(),
                                         ['codex'] if selected == 'codex' else ['codex', 'claude'])
                        self.assertEqual(metadata['verdict'], 'PASS' if selected else 'INCOMPLETE')
                        self.assertEqual(metadata['specialists_completed'], 0)
                        self.assertEqual(metadata['consolidator_completed'], selected is not None)
                        self.assertIn('`specialists_completed=0`', saved)
                        self.assertIn(f'`consolidator_completed={selected is not None}`', saved)
                        self.assertIsNone(metadata.get('cost'))
                        self.assertEqual(metadata['adversarial_model'], selected)
                        self.assertEqual(metadata['requested_model'], 'requested')
                        self.assertIn('**Requested model:** `requested`', saved)
                        self.assertIn(f"**Verdict:** {metadata['verdict']}", saved)
                        if selected:
                            model = 'verified-preferred' if selected == 'codex' else None
                            if case == 'verified-fallback':
                                model = 'verified-second'
                            self.assertEqual(metadata['effective_model'], model)
                            self.assertIn(f'**Adversarial model:** `{selected}`', saved)
                            self.assertIn(f"**Verified effective model:** `{model or 'unknown'}`", saved)
                            if selected == 'claude':
                                self.assertIn('codex: invalid structured review', metadata['fallback_reason'])
                                self.assertIn('using claude CLI', metadata['fallback_reason'])
                                self.assertIn(metadata['fallback_reason'], saved)
                            self.assertIn('Review supplied by ' + selected, saved)
                            sidecar = Path(metadata['retained_decisions']).read_bytes()
                            self.assertEqual(hashlib.sha256(sidecar).hexdigest(), metadata['decisions_sha256'])
                            self.assertIn(metadata['decisions_sha256'], saved)
                            self.assertEqual([os.fsencode(p) for p in json.loads(sidecar)['reviewed_files']],
                                             [filename] + ([b'source'] if selected == 'claude' else []))
                        else:
                            self.assertNotIn('retained_decisions', metadata)
                        if case in ('fallback', 'verified-fallback', 'both-malformed', 'different-byte'):
                            reason = 'coverage' if case == 'different-byte' else 'utf-8'
                            self.assertIn(reason, metadata['fallback_reason'].lower())
                            self.assertIn(reason, saved.lower())

    @unittest.skipIf(os.name == 'nt', 'uses executable fixtures and process groups')
    def test_cli_selection_enforces_each_stream_limit_per_attempt(self):
        # B087 F3 (R48): 8 MiB raw bytes per stream per attempt, fixed here
        # independently of the reader constant.
        limit = 8 * 1024 * 1024
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, _, scope_file = scope_fixture(root)
            base = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=repo, text=True).strip()
            (repo / 'changed.txt').write_text('review this change')
            subprocess.run(['git', 'add', 'changed.txt'], cwd=repo, check=True)
            subprocess.run(['git', 'commit', '-qm', 'change file'], cwd=repo, check=True)
            capture = subprocess.check_output(['bash', str(HERE / 'get_diff.sh')], cwd=repo,
                                              env={**os.environ, 'REVIEW_BASE': base})
            scope_file.write_bytes(capture)
            scope = json.loads(capture)
            prompt = root / 'prompt'
            prompt.write_text('Inspect the frozen changes.')
            attempts = root / 'attempts'

            def reviewer(cli, code=0, out_extra=0, err=0, malformed=False):
                packet = review_packet(scope)
                packet['report'] += '\nReview supplied by ' + cli
                if cli == 'claude':
                    packet['decisions']['reviewed_files'].append('source')
                wire = json.dumps(packet).encode()
                if malformed:
                    wire = b'caf\xe9'
                wire += b' ' * (limit - len(wire) + out_extra) if out_extra is not None else b''
                path = root / cli
                path.write_text(
                    '#!/usr/bin/env python3\nimport sys\n'
                    'assert open("changed.txt", "rb").read() == b"review this change"\n'
                    + ('assert open("source").read() == "new"\n' if cli == 'claude' else '')
                    + f'open({str(attempts)!r}, "a").write({cli!r} + "\\n")\n'
                    + f'sys.stdout.buffer.write({wire!r}); sys.stdout.flush()\n'
                    + f'sys.stderr.buffer.write(b"\\xe9" * {err}); sys.stderr.flush()\n'
                    + f'sys.exit({code})\n')
                path.chmod(0o700)
                return path

            def run(name, preferred, fallback):
                attempts.write_text('')
                report = root / f'{name}.md'
                missing = root / 'missing-claude'
                result = subprocess.run([sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                    '--project-root', str(repo), '--source-root', str(repo), '--base-commit', base,
                    '--scope-file', str(scope_file), '--diff-path', scope['diff_path'],
                    '--prompt-file', str(prompt), '--review-output-path', str(report),
                    '--no-workflow-available', '--no-agent-dispatch-available', '--host', 'claude',
                    '--requested-model', 'R', '--effective-model', 'verified-P',
                    '--fallback-effective-model', 'verified-F',
                    '--codex-command', str(preferred),
                    '--claude-command', str(fallback if fallback else missing)],
                    capture_output=True, text=True)
                return result, report

            def assert_success(name, result, report, selected, calls):
                self.assertEqual(result.returncode, 0, name + result.stderr)
                metadata = json.loads(result.stdout)
                saved = report.read_text()
                self.assertEqual(attempts.read_text().splitlines(), calls, name)
                self.assertEqual(metadata['verdict'], 'PASS', name)
                self.assertEqual(metadata['adversarial_model'], selected, name)
                self.assertEqual(metadata['effective_model'], 'verified-P' if selected == 'codex' else 'verified-F', name)
                self.assertEqual(metadata['requested_model'], 'R')
                self.assertEqual(metadata['specialists_completed'], 0)
                self.assertTrue(metadata['consolidator_completed'])
                self.assertIn('Review supplied by ' + selected, saved)
                self.assertNotIn('Review supplied by ' + ('claude' if selected == 'codex' else 'codex'), saved)
                sidecar = Path(metadata['retained_decisions']).read_bytes()
                self.assertEqual(hashlib.sha256(sidecar).hexdigest(), metadata['decisions_sha256'], name)
                self.assertIn(metadata['decisions_sha256'], saved)
                reviewed = json.loads(sidecar)['reviewed_files']
                self.assertEqual(('source' in reviewed), selected == 'claude', name)

            def assert_exhausted(name, result, report, calls, reasons):
                self.assertEqual(result.returncode, 2, name + result.stderr)
                metadata = json.loads(result.stdout)
                saved = report.read_text()
                self.assertEqual(attempts.read_text().splitlines(), calls, name)
                self.assertEqual(metadata['verdict'], 'INCOMPLETE', name)
                self.assertEqual(metadata['requested_model'], 'R')
                self.assertIsNone(metadata['effective_model'])
                self.assertEqual(metadata['specialists_completed'], 0)
                self.assertFalse(metadata['consolidator_completed'])
                self.assertNotIn('retained_decisions', metadata)
                self.assertNotIn('decisions_sha256', metadata)
                self.assertIn('INCOMPLETE', saved)
                positions = [metadata['fallback_reason'].find(r) for r in reasons]
                self.assertTrue(all(p >= 0 for p in positions), (name, reasons, metadata['fallback_reason']))
                self.assertEqual(positions, sorted(positions), name)
                self.assertNotIn('Review supplied by', saved)

            # Stderr limit, independently for the preferred attempt.
            result, report = run('p-at', reviewer('codex', err=limit), reviewer('claude'))
            assert_success('p-at', result, report, 'codex', ['codex'])
            result, report = run('p-over', reviewer('codex', err=limit + 1), reviewer('claude'))
            assert_success('p-over', result, report, 'claude', ['codex', 'claude'])
            # Stdout limit, independently of stderr; both streams exactly full is eligible.
            result, report = run('both-at', reviewer('codex', err=limit), reviewer('claude'))
            assert_success('both-at', result, report, 'codex', ['codex'])
            result, report = run('out-over', reviewer('codex', out_extra=1, err=limit), reviewer('claude'))
            assert_success('out-over', result, report, 'claude', ['codex', 'claude'])
            result, report = run('err-over-only', reviewer('codex', err=limit + 1), reviewer('claude'))
            assert_success('err-over-only', result, report, 'claude', ['codex', 'claude'])
            # Fixed preferred failures, then a fallback at and over each limit.
            for label, failing_kwargs in (('exit7', {'code': 7}), ('malformed', {'malformed': True})):
                failing = reviewer('codex', **failing_kwargs)
                reason = 'non-zero exit status 7' if label == 'exit7' else 'invalid structured review'
                result, report = run(label + '-at', failing, reviewer('claude', err=limit))
                assert_success(label + '-at', result, report, 'claude', ['codex', 'claude'])
                self.assertIn(reason, json.loads(result.stdout)['fallback_reason'])
                result, report = run(label + '-fb-err', failing, reviewer('claude', err=limit + 1))
                assert_exhausted(label + '-fb-err', result, report, ['codex', 'claude'], [reason, 'claude: non-zero exit status 1'])
                result, report = run(label + '-fb-out', failing, reviewer('claude', out_extra=1))
                assert_exhausted(label + '-fb-out', result, report, ['codex', 'claude'], [reason, 'claude: non-zero exit status 1'])
            # Both attempts over their limit are recorded in order.
            result, report = run('both-fail', reviewer('codex', err=limit + 1), reviewer('claude', err=limit + 1))
            assert_exhausted('both-fail', result, report, ['codex', 'claude'],
                             ['codex: non-zero exit status 1', 'claude: non-zero exit status 1'])
            # A fallback after preferred exit 7 starts with a full allowance of its own.
            result, report = run('fresh-allowance', reviewer('codex', code=7, out_extra=0, err=limit),
                                 reviewer('claude', out_extra=0, err=limit))
            assert_success('fresh-allowance', result, report, 'claude', ['codex', 'claude'])
            # Only one eligible executable.
            result, report = run('solo-at', reviewer('codex', err=limit), None)
            assert_success('solo-at', result, report, 'codex', ['codex'])
            result, report = run('solo-over', reviewer('codex', err=limit + 1), None)
            assert_exhausted('solo-over', result, report, ['codex'], ['codex: non-zero exit status 1'])

    @unittest.skipIf(os.name == 'nt', 'uses executable fixtures')
    def test_cli_attributes_each_post_selection_scope_failure(self):
        # B087 F3 (R46): after a reviewer reads candidate A, the checkout may move to B
        # while A's capture stays readable. Each cause needs its own diagnosis.
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            base = scope['base_commit']
            prompt = root / 'prompt'
            prompt.write_text('Inspect the frozen changes.')
            reviewer = root / 'reviewer'
            mutation = root / 'mutation'

            def run(name, action):
                mutation.write_text(action)
                packet = review_packet(scope)
                reviewer.write_text(
                    '#!/usr/bin/env python3\nimport subprocess, sys\n'
                    'assert open("source").read() == "new"\n'
                    f'sys.stdout.write({json.dumps(packet)!r})\n'
                    f'exec(open({str(mutation)!r}).read())\n')
                reviewer.chmod(0o700)
                report = root / f'{name}.md'
                result = subprocess.run([sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                    '--project-root', str(repo), '--source-root', str(repo), '--base-commit', base,
                    '--scope-file', str(scope_file), '--diff-path', scope['diff_path'],
                    '--prompt-file', str(prompt), '--review-output-path', str(report),
                    '--no-workflow-available', '--no-agent-dispatch-available', '--host', 'claude',
                    '--requested-model', 'R', '--effective-model', 'verified',
                    '--codex-command', str(reviewer), '--claude-command', str(root / 'absent')],
                    capture_output=True, text=True)
                return result, report

            def git(*args):
                subprocess.run(['git', '-c', 'commit.gpgsign=false', '-c', 'core.hooksPath=/dev/null', *args],
                               cwd=repo, check=True, capture_output=True)

            def assert_incomplete(name, result, report, reason):
                self.assertEqual(result.returncode, 2, name + result.stderr)
                metadata = json.loads(result.stdout)
                self.assertEqual(metadata['verdict'], 'INCOMPLETE', name)
                self.assertEqual(metadata['specialists_completed'], 0)
                self.assertFalse(metadata['consolidator_completed'])
                self.assertNotIn('retained_decisions', metadata)
                self.assertIn(reason, metadata['fallback_reason'], name)
                self.assertIn(reason, report.read_text(), name)

            result, report = run('a', 'pass\n')
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(json.loads(result.stdout)['verdict'], 'PASS')

            result, report = run('missing', f'import os; os.rename({str(scope_file)!r}, {str(scope_file) + ".gone"!r})\n')
            assert_incomplete('missing', result, report, 'scope')
            self.assertNotIn('candidate changed since scope capture', json.loads(result.stdout)['fallback_reason'])
            os.rename(str(scope_file) + '.gone', scope_file)

            # Checkout moves to B with A's scope and diff still readable and byte-identical.
            drift = ('import subprocess\n'
                     f'subprocess.run(["git", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "commit", "--allow-empty", "-qm", "B"], cwd={str(repo)!r}, check=True)\n')
            head = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=repo, text=True).strip()
            result, report = run('drift', drift)
            assert_incomplete('drift', result, report, 'candidate changed since scope capture')
            self.assertNotEqual(subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=repo, text=True).strip(), head)
            git('reset', '-q', '--hard', head)

            # Restored control on the original capture passes again.
            result, report = run('restored', 'pass\n')
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(json.loads(result.stdout)['verdict'], 'PASS')

    @unittest.skipIf(os.name == 'nt', 'uses shell executable fixture')
    def test_scope_timeout_produces_structured_failure_and_report(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            fake_bin = root / 'bin'
            fake_bin.mkdir()
            git = fake_bin / 'git'
            git.write_text('#!/bin/sh\nsleep 3\n')
            git.chmod(0o700)
            env = {**os.environ, 'PATH': str(fake_bin) + os.pathsep + os.environ['PATH'], 'SHARK_REVIEW_GIT_TIMEOUT': '0.2'}
            script = str(HERE / 'adaptive_review.py')
            validate = subprocess.run([sys.executable, script, 'validate-scope', '--scope-file', str(scope_file), '--project-root', str(repo), '--source-root', str(repo)], env=env, capture_output=True, text=True)
            self.assertEqual(validate.returncode, 2, validate.stderr)
            self.assertFalse(json.loads(validate.stdout)['valid'])
            body = root / 'body'
            body.write_text('Reviewed scope: all files and callers. Findings: none. Detailed verification completed. VERDICT: PASS')
            report = root / 'report.md'
            writer = subprocess.run([sys.executable, script, 'write-report', '--body-file', str(body), '--project-root', str(repo), '--source-root', str(repo), '--review-output-path', str(report), '--runner-mode', 'adversarial-cli', '--specialists-completed', '0', '--consolidator-completed', 'true', '--fallback-reason', 'native unavailable', '--adversarial-model', 'codex', '--base-commit', scope['base_commit'], '--diff-path', scope['diff_path'], '--verdict', 'PASS', '--scope-file', str(scope_file)], env=env, capture_output=True, text=True)
            self.assertEqual(writer.returncode, 2, writer.stderr)
            self.assertEqual(json.loads(writer.stdout)['verdict'], 'INCOMPLETE')
            self.assertTrue(report.exists())
            fake_shark = fake_bin / 'shark'
            fake_shark.write_text('#!/bin/sh\nsleep 3\n')
            fake_shark.chmod(0o700)
            env.update({'SHARK_REVIEW_IO_FORCE': '1', 'SHARK_REVIEW_INVENTORY_BIN': str(fake_shark)})
            second_report = root / 'io-timeout-report.md'
            args = writer.args[:]
            args[args.index('--review-output-path') + 1] = str(second_report)
            failed_io = subprocess.run(args, env=env, capture_output=True, text=True)
            self.assertEqual(failed_io.returncode, 2, failed_io.stderr)
            self.assertEqual(json.loads(failed_io.stdout)['verdict'], 'INCOMPLETE')
            self.assertTrue(second_report.exists())

    def test_failed_report_can_describe_product_finding(self):
        body = ('CURRENT_REVIEW_STATUS: COMPLETE\n'
                'FINDING_CONTEXT: application external input validation is deficient.\n'
                'Scope: all candidate files reviewed. Findings: validation bug. VERDICT: FAIL\n'
                'FINDING_DETAIL: INCOMPLETE VALIDATION | F-1 | non-blocker | triage')
        self.assertTrue(adaptive_review.valid_review_report(body, 'FAIL'))
        self.assertFalse(adaptive_review.valid_review_report(body + '\nThe current review is incomplete.', 'FAIL'))

    def test_report_finding_requires_a_retained_decision(self):
        body = ('CURRENT_REVIEW_STATUS: COMPLETE\n'
                'Scope: all candidate files reviewed. Findings: an issue needs triage. VERDICT: PASS-WITH-TRIAGE\n'
                'FINDING_DETAIL: INCOMPLETE PRODUCT BEHAVIOR | F-1 | non-blocker | triage')
        self.assertTrue(adaptive_review.valid_review_report(body, 'PASS-WITH-TRIAGE'))
        with self.assertRaisesRegex(ValueError, 'retained decision'):
            adaptive_review._require_finding_decision(body, {'findings': []})
        with self.assertRaisesRegex(ValueError, 'retained decision'):
            adaptive_review._require_finding_decision(body, {'findings': [{'finding_id': 'F-2', 'severity': 'non-blocker', 'disposition': 'triage'}]})
        with self.assertRaisesRegex(ValueError, 'retained decision'):
            adaptive_review._require_finding_decision(body, {'findings': [{'finding_id': 'F-1', 'severity': 'non-blocker', 'disposition': 'refuted'}]})
        adaptive_review._require_finding_decision(body, {'findings': [{'finding_id': 'F-1', 'severity': 'non-blocker', 'disposition': 'triage'}]})
        with self.assertRaisesRegex(ValueError, 'retained decision'):
            adaptive_review._require_finding_decision(body, {'findings': [{'finding_id': 'F-1', 'severity': 'non-blocker', 'disposition': 'Triage'}]})
        multi = body + '\nHISTORICAL_REVIEW_STATUS: DID_NOT_FINISH\nFINDING_DETAIL: INCOMPLETE VALIDATION | F-2 | nit | report-only'
        decisions = {'findings': [{'finding_id': 'F-1', 'severity': 'non-blocker', 'disposition': 'triage'}, {'finding_id': 'F-2', 'severity': 'nit', 'disposition': 'report-only'}]}
        self.assertTrue(adaptive_review.valid_review_report(multi, 'PASS-WITH-TRIAGE'))
        adaptive_review._require_finding_decision(multi, decisions)
        with self.assertRaisesRegex(ValueError, 'unique'):
            adaptive_review._require_finding_decision(body + '\nFINDING_DETAIL: INCOMPLETE VALIDATION | F-1 | non-blocker | triage', decisions)

    def test_nit_decisions_are_report_only(self):
        finding = {'file': 'app.go', 'line': 2, 'severity': 'nit', 'rule': 'STYLE',
                   'summary': 'label', 'diagnosis': 'unclear label', 'disposition': 'triage',
                   'reason': 'owner decision requires no action'}
        with self.assertRaisesRegex(ValueError, 'nit disposition must be report-only'):
            adaptive_review._validate_decision(finding, 'PASS-WITH-TRIAGE')
        finding['disposition'] = 'report-only'
        adaptive_review._validate_decision(finding, 'PASS-WITH-TRIAGE')


    def test_crlf_footer_requires_matching_decision_and_nonclean_verdict(self):
        scope = {'candidate_commit': 'commit', 'candidate_tree': 'tree', 'diff_sha256': 'diff', 'changed_files': ['app.go', 'other.go']}
        identity = {'commit': 'commit', 'tree': 'tree', 'diff_sha256': 'diff'}
        body = ('CURRENT_REVIEW_STATUS: COMPLETE\r\n'
                'Scope: all candidate files reviewed. Findings: input defect. VERDICT: PASS\r\n'
                'FINDING_DETAIL: INCOMPLETE VALIDATION | F-1 | non-blocker | triage\r\n')
        packet = {'report': body, 'decisions': {'candidate_identity': identity, 'reviewed_files': ['app.go', 'other.go'], 'findings': []}}
        with self.assertRaises(ValueError):
            adaptive_review._parse_cli_packet(json.dumps(packet), scope)
        packet['report'] = body.replace('VERDICT: PASS', 'VERDICT: PASS-WITH-TRIAGE')
        with self.assertRaises(ValueError):
            adaptive_review._parse_cli_packet(json.dumps(packet), scope)
        packet['decisions']['findings'] = [{'finding_id': 'F-1', 'file': 'app.go', 'line': 2,
            'severity': 'non-blocker', 'rule': 'INPUT', 'summary': 'validation',
            'diagnosis': 'missing check', 'disposition': 'triage', 'reason': 'tracked'}]
        self.assertEqual(adaptive_review._parse_cli_packet(json.dumps(packet), scope)[0], packet['report'])
        packet['report'] += 'FINDING_DETAIL: INCOMPLETE PRODUCT BEHAVIOR | F-2 | nit | report-only\r\n'
        with self.assertRaises(ValueError):
            adaptive_review._parse_cli_packet(json.dumps(packet), scope)
        packet['decisions']['findings'].append({'finding_id': 'F-2', 'file': 'other.go', 'line': 3,
            'severity': 'nit', 'rule': 'STYLE', 'summary': 'other issue',
            'diagnosis': 'incorrect label', 'disposition': 'report-only', 'reason': 'owner decision requires no action'})
        self.assertEqual(adaptive_review._parse_cli_packet(json.dumps(packet), scope)[0], packet['report'])

    def test_capability_ladder(self):
        self.assertEqual(select_runner(Capabilities(True, True, True, True))['runner_mode'], 'canonical-six-angle')
        self.assertEqual(select_runner(Capabilities(False, True, True, True))['runner_mode'], 'dispatched-six-angle')
        self.assertEqual(select_runner(Capabilities(False, False, True, False), 'claude')['adversarial_model'], 'codex')
        self.assertEqual(select_runner(Capabilities(False, False, False, True), 'codex')['adversarial_model'], 'claude')
        self.assertEqual(select_runner(Capabilities(False, False, True, False), 'codex')['adversarial_model'], 'codex')
        self.assertEqual(select_runner(Capabilities(False, False, False, False))['runner_mode'], 'incomplete')

    def test_cli_candidates_retry_host_cli_after_preferred_alternate(self):
        self.assertEqual(
            cli_candidates(Capabilities(False, False, True, True), 'codex', 'claude'),
            ['claude', 'codex'],
        )

    def test_output_validation_rejects_partial(self):
        good = 'REVIEWED SCOPE: 2 files\nFINDINGS: none\nThe change is safe and complete.\nVERDICT: PASS'
        self.assertTrue(valid_adversarial_output(good))
        for bad in ('', 'VERDICT: PASS', 'authentication failed', 'REVIEWED SCOPE\nFINDINGS'):
            self.assertFalse(valid_adversarial_output(bad))

    def test_review_decisions_require_explicit_refutation(self):
        scope = {'candidate_commit': 'c', 'candidate_tree': 't', 'diff_sha256': 'd', 'changed_files': ['a.go']}
        blocker = {'file': 'a.go', 'line': 3, 'summary': 'broken', 'severity': 'blocker', 'rule': 'BUG', 'diagnosis': 'wrong'}
        decisions = {'candidate_identity': {'commit': 'c', 'tree': 't', 'diff_sha256': 'd'}, 'reviewed_files': ['a.go'], 'findings': []}
        with self.assertRaises(ValueError):
            validate_review_decisions(decisions, scope, [blocker], 'PASS')
        decisions['findings'] = [{**blocker, 'disposition': 'open', 'reason': 'not refuted'}]
        with self.assertRaises(ValueError):
            validate_review_decisions(decisions, scope, [blocker], 'PASS')
        decisions['findings'][0]['disposition'] = 'refuted'
        decisions['findings'][0]['reason'] = 'evidence shows the caller cannot reach this branch'
        validate_review_decisions(decisions, scope, [blocker], 'PASS')

    def test_failed_review_keeps_every_raw_finding(self):
        scope = {'candidate_commit': 'c', 'candidate_tree': 't', 'diff_sha256': 'd', 'changed_files': ['a.go']}
        finding = {'file': 'a.go', 'line': 3, 'summary': 'broken', 'severity': 'blocker', 'rule': 'BUG', 'diagnosis': 'wrong'}
        decisions = {'candidate_identity': {'commit': 'c', 'tree': 't', 'diff_sha256': 'd'}, 'reviewed_files': ['a.go'], 'findings': []}
        with self.assertRaisesRegex(ValueError, 'no explicit disposition'):
            validate_review_decisions(decisions, scope, [finding], 'FAIL')
        decisions['findings'] = [{**finding, 'disposition': 'open', 'reason': 'still reproducible'}]
        validate_review_decisions(decisions, scope, [finding], 'FAIL')

    def test_cli_decisions_reject_omitted_manifest_file(self):
        scope = {'candidate_commit': 'c', 'candidate_tree': 't', 'diff_sha256': 'd', 'changed_files': ['one.go', 'two.go']}
        decisions = {'candidate_identity': {'commit': 'c', 'tree': 't', 'diff_sha256': 'd'}, 'reviewed_files': ['one.go'], 'findings': []}
        with self.assertRaisesRegex(ValueError, 'incomplete reviewed_files coverage'):
            validate_review_decisions(decisions, scope, [], 'PASS', require_reviewed_files=True)
        decisions['reviewed_files'].append('two.go')
        validate_review_decisions(decisions, scope, [], 'PASS', require_reviewed_files=True)

    def test_supplemental_context_must_be_a_repository_relative_path(self):
        with tempfile.TemporaryDirectory() as tmp:
            _, scope, _ = scope_fixture(Path(tmp))
            packet = review_packet(scope)
            for path in ('/outside.go', '../outside.go', 'x/../../outside.go'):
                with self.subTest(path=path):
                    packet['decisions']['reviewed_files'] = [*scope['changed_files'], path]
                    with self.assertRaisesRegex(ValueError, 'repository-relative') as caught:
                        adaptive_review._parse_cli_packet(json.dumps(packet), scope)
                    self.assertIn(path, str(caught.exception))

    def test_report_retains_validated_bytes_after_source_replacement(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            identity = review_packet(scope)['decisions']['candidate_identity']
            angles = root / 'angles.json'
            angles.write_text(json.dumps({name: {'candidate_identity': identity, 'reviewed_files': scope['changed_files'], 'findings': []} for name in 'abcdef'}))
            decisions = root / 'decisions.json'
            decisions.write_text(json.dumps({'candidate_identity': identity, 'findings': []}))
            body = root / 'body'
            body.write_text(review_packet(scope)['report'])
            report = root / 'report'
            diff = Path(scope['diff_path'])
            expected = {str(path): path.read_bytes() for path in (angles, diff)}
            original_read = adaptive_review.read_review_file
            def replace_after_read(path, limit):
                data = original_read(path, limit)
                if str(path) in expected:
                    Path(path).write_bytes(b'replaced')
                return data
            args = argparse.Namespace(runner_mode='dispatched-six-angle', specialists_completed=6, consolidator_completed=True,
                adversarial_model=None, fallback_reason='native Workflow unavailable', base_commit=scope['base_commit'], diff_path=str(diff), verdict='PASS',
                requested_model=None, effective_model=None, scope_file=str(scope_file), project_root=str(repo), source_root=str(repo),
                shark_project_root=None, angle_results=str(angles), review_decisions=str(decisions), body_file=str(body),
                review_output_path=str(report), branch='fixture')
            with patch('adaptive_review.read_review_file', side_effect=replace_after_read), contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(adaptive_review.write_report(args), 0)
            self.assertEqual(Path(str(report) + '.angles.json').read_bytes(), expected[str(angles)])
            self.assertEqual(Path(str(report) + '.diff').read_bytes(), expected[str(diff)])

    def test_retained_artifacts_reject_output_above_shared_limit(self):
        with tempfile.TemporaryDirectory() as tmp, patch('adaptive_review.MAX_ARTIFACT_BYTES', 4):
            path = Path(tmp) / 'evidence'
            with self.assertRaisesRegex(ValueError, 'exceeds size limit'):
                adaptive_review._write_new_bytes(path, b'12345')
            self.assertFalse(path.exists())

    def test_cli_success_persists_metadata(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            prompt = root / 'prompt.txt'
            prompt.write_text('review')
            fake = root / 'fake-reviewer'
            fake_reviewer(fake, review_packet(scope))
            report = root / 'report.md'
            proc = subprocess.run([
                sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                '--prompt-file', str(prompt), '--project-root', str(repo), '--source-root', str(repo), '--diff-path', scope['diff_path'],
                '--scope-file', str(scope_file), '--base-commit', scope['base_commit'], '--review-output-path', str(report), '--host', 'claude',
                '--codex-command', str(fake), '--timeout', '2', '--no-workflow-available', '--no-agent-dispatch-available',
            ], capture_output=True, text=True, check=False)
            self.assertEqual(proc.returncode, 0, proc.stderr)
            metadata = json.loads(proc.stdout)
            self.assertEqual(metadata['runner_mode'], 'adversarial-cli')
            self.assertIn('Base commit:', report.read_text())
            self.assertIn(scope['base_commit'], report.read_text())
            self.assertIn('Adversarial model:', report.read_text())
            self.assertIn('`codex`', report.read_text())

    def test_cli_writer_requires_model_and_fallback_reason(self):
        for model, reason in [('', 'unavailable'), ('unknown', 'unavailable'), ('codex', ''), ('codex', 'none'), ('codex', '`unknown`'), ('codex', '` none `')]:
            args = argparse.Namespace(runner_mode='adversarial-cli', specialists_completed=0,
                                      consolidator_completed=True, adversarial_model=model, fallback_reason=reason, verdict='PASS')
            metadata = {'runner_mode': args.runner_mode, 'verdict': 'PASS', 'fallback_reason': reason}
            self.assertTrue(adaptive_review._writer_runner_state(args, metadata, None, False))
            self.assertEqual(metadata['verdict'], 'INCOMPLETE')
        args = argparse.Namespace(runner_mode='dispatched-six-angle', specialists_completed=6,
                                  consolidator_completed=True, adversarial_model=None, fallback_reason=None, verdict='PASS')
        metadata = {'runner_mode': args.runner_mode, 'verdict': 'PASS', 'fallback_reason': None}
        self.assertTrue(adaptive_review._writer_runner_state(args, metadata, None, False))
        self.assertEqual(metadata['verdict'], 'INCOMPLETE')

    def test_writer_rejects_unsupported_verdict(self):
        for verdict in ('BANANA', 'INCOMPLETE', ''):
            args = argparse.Namespace(runner_mode='adversarial-cli', specialists_completed=0,
                                      consolidator_completed=True, adversarial_model='codex',
                                      fallback_reason='native review unavailable', verdict=verdict)
            metadata = {'runner_mode': args.runner_mode, 'verdict': verdict, 'fallback_reason': args.fallback_reason}
            self.assertTrue(adaptive_review._writer_runner_state(args, metadata, None, False))
            self.assertEqual(metadata['verdict'], 'INCOMPLETE')
            self.assertEqual(metadata['runner_mode'], 'incomplete')

    def test_final_scope_must_remain_readable_and_identical(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            prompt = root / 'prompt.txt'
            prompt.write_text('review')
            reviewer = root / 'reviewer'
            changed = {**scope, 'repair_plan_sha256': 'a' * 64}
            for name, final_bytes, diagnostic in (
                ('unreadable', None, 'scope validation failed'),
                ('changed', json.dumps(changed), 'captured scope changed'),
                ('identical', json.dumps(scope), None),
            ):
                with self.subTest(name=name):
                    scope_file.write_text(json.dumps(scope))
                    reviewer.write_text(
                        '#!/usr/bin/env python3\nimport json\nfrom pathlib import Path\n'
                        f'path = Path({str(scope_file)!r})\n'
                        + ('path.unlink()\n' if final_bytes is None else f'path.write_text({final_bytes!r})\n')
                        + f'print(json.dumps({review_packet(scope)!r}))\n'
                    )
                    reviewer.chmod(0o700)
                    report = root / name
                    result = subprocess.run([
                        sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                        '--prompt-file', str(prompt), '--project-root', str(repo), '--source-root', str(repo),
                        '--diff-path', scope['diff_path'], '--scope-file', str(scope_file),
                        '--base-commit', scope['base_commit'], '--review-output-path', str(report),
                        '--host', 'claude', '--codex-command', str(reviewer), '--no-workflow-available',
                        '--no-agent-dispatch-available',
                    ], capture_output=True, text=True)
                    metadata = json.loads(result.stdout)
                    self.assertEqual(result.returncode, 2 if diagnostic else 0)
                    self.assertEqual(metadata['verdict'], 'INCOMPLETE' if diagnostic else 'PASS')
                    self.assertEqual(metadata['specialists_completed'], 0)
                    self.assertEqual(metadata['consolidator_completed'], diagnostic is None)
                    saved = report.read_text()
                    self.assertIn(f'`specialists_completed=0` `consolidator_completed={diagnostic is None}`', saved)
                    self.assertIn(f"**Verdict:** {metadata['verdict']}", saved)
                    if diagnostic:
                        self.assertIn(diagnostic, metadata['fallback_reason'])
                        self.assertIn(diagnostic, saved)

    @unittest.skipIf(os.name == 'nt', 'uses shell executable fixture')
    def test_cli_inspection_must_cover_the_frozen_manifest(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, _, scope_file = scope_fixture(root)
            (repo / 'second').write_text('second changed file')
            subprocess.run(['git', 'add', 'second'], cwd=repo, check=True)
            subprocess.run(['git', 'commit', '-qm', 'add second file'], cwd=repo, check=True)
            capture = subprocess.check_output(['bash', str(HERE / 'get_diff.sh')], cwd=repo)
            scope_file.write_bytes(capture)
            scope = json.loads(capture)
            self.assertEqual(set(scope['changed_files']), {'source', 'second'})
            prompt = root / 'prompt'
            prompt.write_text('Inspect every changed file: source and second.')
            for inspected, expected in [(['source', 'second'], 'PASS'), (['source'], 'INCOMPLETE')]:
                with self.subTest(inspected=inspected):
                    reviewer = root / 'reviewer'
                    reviewer.write_text(
                        '#!/usr/bin/env python3\nfrom pathlib import Path\nimport json, sys\n'
                        'assert "decisions.reviewed_files must list every frozen changed file" in sys.argv[-1]\n'
                        f'paths = {inspected!r}\npacket = {review_packet(scope)!r}\n'
                        'for path in paths: Path(path).read_text()\n'
                        'packet["decisions"]["reviewed_files"] = paths\n'
                        'packet["report"] = ("Scope inspected: " + ", ".join(paths) + '
                        '"\\nChecks: read each listed file and checked its contents.'
                        '\\nFindings: 0 defects found.\\nVerdict: PASS")\n'
                        'print(json.dumps(packet))\n'
                    )
                    reviewer.chmod(0o700)
                    report = root / f'report-{len(inspected)}.md'
                    result = subprocess.run([
                        sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                        '--prompt-file', str(prompt), '--project-root', str(repo), '--source-root', str(repo),
                        '--diff-path', scope['diff_path'], '--scope-file', str(scope_file),
                        '--base-commit', scope['base_commit'], '--review-output-path', str(report),
                        '--host', 'claude', '--codex-command', str(reviewer), '--claude-command', str(reviewer),
                        '--no-workflow-available', '--no-agent-dispatch-available',
                    ], capture_output=True, text=True)
                    metadata = json.loads(result.stdout)
                    self.assertEqual(metadata['verdict'], expected, report.read_text())
                    self.assertEqual(result.returncode, 0 if expected == 'PASS' else 2)

    @unittest.skipIf(os.name == 'nt', 'uses shell executable fixture')
    def test_run_cli_verdict_matrix_requires_every_frozen_file_and_retains_valid_decisions(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, _, scope_file = scope_fixture(root)
            (repo / 'second').write_text('second changed file')
            subprocess.run(['git', 'add', 'second'], cwd=repo, check=True)
            subprocess.run(['git', 'commit', '-qm', 'add second file'], cwd=repo, check=True)
            scope_bytes = subprocess.check_output(['bash', str(HERE / 'get_diff.sh')], cwd=repo)
            scope_file.write_bytes(scope_bytes)
            scope = json.loads(scope_bytes)
            self.assertEqual(set(scope['changed_files']), {'source', 'second'})
            prompt = root / 'prompt'
            prompt.write_text('Inspect the frozen manifest and relevant callers.')
            for verdict in ('PASS', 'PASS-WITH-TRIAGE', 'FAIL'):
                for omitted in (False, True):
                    with self.subTest(verdict=verdict, omitted=omitted):
                        packet = verdict_packet(scope, verdict)
                        if omitted:
                            packet['decisions']['reviewed_files'] = [scope['changed_files'][0]]
                        reviewer = root / 'reviewer'
                        reviewer.write_text(
                            '#!/usr/bin/env python3\nfrom pathlib import Path\nimport json, sys\n'
                            'assert "decisions.reviewed_files must list every frozen changed file" in sys.argv[-1]\n'
                            f'packet = json.loads({json.dumps(json.dumps(packet))})\n'
                            'for path in packet["decisions"]["reviewed_files"]: Path(path).read_text()\n'
                            'print(json.dumps(packet))\n'
                        )
                        reviewer.chmod(0o700)
                        report = root / f'run-cli-{verdict}-{omitted}.md'
                        result = subprocess.run([
                            sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                            '--prompt-file', str(prompt), '--project-root', str(repo), '--source-root', str(repo),
                            '--diff-path', scope['diff_path'], '--scope-file', str(scope_file),
                            '--base-commit', scope['base_commit'], '--review-output-path', str(report),
                            '--host', 'claude', '--codex-command', str(reviewer), '--claude-command', str(reviewer),
                            '--no-workflow-available', '--no-agent-dispatch-available',
                        ], capture_output=True, text=True)
                        metadata = json.loads(result.stdout)
                        expected = 'INCOMPLETE' if omitted else verdict
                        self.assertEqual(metadata['verdict'], expected, result.stdout + result.stderr)
                        self.assertEqual(metadata['runner_mode'], 'incomplete' if omitted else 'adversarial-cli')
                        self.assertEqual(result.returncode, 2 if omitted else 0)
                        saved_report = report.read_text()
                        self.assertIn(f'**Verdict:** {expected}', saved_report)
                        self.assertIn(f'`runner_mode={metadata["runner_mode"]}`', saved_report)
                        if omitted:
                            current_body = saved_report.split("---", 1)[1]
                            self.assertIn('CURRENT_REVIEW_STATUS: INCOMPLETE', current_body)
                            self.assertIn('VERDICT: INCOMPLETE', current_body)
                            self.assertNotRegex(current_body, r'CURRENT_REVIEW_STATUS:\s*COMPLETE')
                            self.assertNotRegex(current_body, r'VERDICT\s*:?\s*PASS(?:-WITH-TRIAGE)?\b')
                        if not omitted:
                            decisions_path = Path(metadata['retained_decisions'])
                            decisions_bytes = decisions_path.read_bytes()
                            decisions_digest = hashlib.sha256(decisions_bytes).hexdigest()
                            self.assertEqual(json.loads(decisions_bytes), packet['decisions'])
                            self.assertEqual(metadata['decisions_sha256'], decisions_digest)
                            self.assertRegex(saved_report, rf'Review decisions SHA-256:\*\* `{decisions_digest}`')

    @unittest.skipIf(os.name == 'nt', 'uses shell executable fixture')
    def test_public_writer_rejects_angle_finding_from_unreviewed_path(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, _, scope_file = scope_fixture(root)
            (repo / 'second').write_text('second changed file')
            subprocess.run(['git', 'add', 'second'], cwd=repo, check=True)
            subprocess.run(['git', 'commit', '-qm', 'add second file'], cwd=repo, check=True)
            scope_bytes = subprocess.check_output(['bash', str(HERE / 'get_diff.sh')], cwd=repo)
            scope_file.write_bytes(scope_bytes)
            scope = json.loads(scope_bytes)
            identity = {'commit': scope['candidate_commit'], 'tree': scope['candidate_tree'],
                        'diff_sha256': scope['diff_sha256']}
            raw_finding = {'file': 'second', 'line': 1, 'severity': 'nit', 'rule': 'STYLE',
                           'summary': 'label', 'diagnosis': 'unclear label'}
            angles = {name: {'candidate_identity': identity, 'reviewed_files': ['source', 'second'], 'findings': []}
                      for name in 'abcdef'}
            angles['a']['findings'] = [raw_finding]
            decisions = {'candidate_identity': identity, 'findings': [{**raw_finding, 'finding_id': 'F-1',
                         'disposition': 'report-only', 'reason': 'owner decision requires no action'}]}
            body = root / 'body'
            body.write_text('CURRENT_REVIEW_STATUS: COMPLETE\nScope: all changed files were inspected. Findings: one report-only nit. '
                            'The review checked candidate evidence and coverage. VERDICT: PASS-WITH-TRIAGE\n'
                            'FINDING_DETAIL: INCOMPLETE PRODUCT BEHAVIOR | F-1 | nit | report-only')
            angle_file, decision_file = root / 'angles.json', root / 'decisions.json'
            angle_file.write_text(json.dumps(angles))
            decision_file.write_text(json.dumps(decisions))
            for omitted, expected in [(False, 'PASS-WITH-TRIAGE'), (True, 'INCOMPLETE')]:
                with self.subTest(omitted=omitted):
                    current = json.loads(angle_file.read_text())
                    if omitted:
                        current['a']['reviewed_files'] = ['source']
                    angle_file.write_text(json.dumps(current))
                    report = root / f'writer-{omitted}.md'
                    result = subprocess.run([
                        sys.executable, str(HERE / 'adaptive_review.py'), 'write-report',
                        '--body-file', str(body), '--project-root', str(repo), '--source-root', str(repo),
                        '--review-output-path', str(report), '--runner-mode', 'dispatched-six-angle',
                        '--specialists-completed', '6', '--consolidator-completed', 'true',
                        '--fallback-reason', 'agent dispatch used for fixture',
                        '--base-commit', scope['base_commit'], '--diff-path', scope['diff_path'],
                        '--verdict', 'PASS-WITH-TRIAGE', '--scope-file', str(scope_file),
                        '--angle-results', str(angle_file), '--review-decisions', str(decision_file),
                    ], capture_output=True, text=True)
                    self.assertEqual(json.loads(result.stdout)['verdict'], expected, result.stdout + result.stderr)
                    self.assertEqual(result.returncode, 0 if not omitted else 2)
                    metadata = json.loads(result.stdout)
                    self.assertEqual(metadata['specialists_completed'], 5 if omitted else 6)
                    self.assertEqual(metadata['consolidator_completed'], not omitted)
                    self.assertIn(f"`specialists_completed={metadata['specialists_completed']}`", report.read_text())
                    self.assertIn(f"`consolidator_completed={not omitted}`", report.read_text())
                    if omitted:
                        self.assertIn('angle a', metadata['fallback_reason'])
                        self.assertIn('second', metadata['fallback_reason'])
                        self.assertIn(metadata['fallback_reason'], report.read_text())
                    if omitted:
                        saved_report = report.read_text().split("---", 1)[1]
                        self.assertIn('CURRENT_REVIEW_STATUS: INCOMPLETE', saved_report)
                        self.assertIn('VERDICT: INCOMPLETE', saved_report)
                        self.assertNotRegex(saved_report, r'CURRENT_REVIEW_STATUS:\s*COMPLETE')
                        self.assertNotRegex(saved_report, r'VERDICT\s*:?\s*PASS(?:-WITH-TRIAGE)?\b')

    @unittest.skipIf(os.name == 'nt', 'uses shell executable fixture')
    def test_public_cli_writer_requires_complete_reviewed_file_manifest(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, _, scope_file = scope_fixture(root)
            (repo / 'second').write_text('second changed file')
            subprocess.run(['git', 'add', 'second'], cwd=repo, check=True)
            subprocess.run(['git', 'commit', '-qm', 'add second file'], cwd=repo, check=True)
            scope_bytes = subprocess.check_output(['bash', str(HERE / 'get_diff.sh')], cwd=repo)
            scope_file.write_bytes(scope_bytes)
            scope = json.loads(scope_bytes)
            decisions_path = root / 'decisions.json'
            for verdict in ('PASS', 'PASS-WITH-TRIAGE', 'FAIL'):
                packet = verdict_packet(scope, verdict)
                for omitted in (False, True):
                    with self.subTest(verdict=verdict, omitted=omitted):
                        decisions = json.loads(json.dumps(packet['decisions']))
                        if omitted:
                            decisions['reviewed_files'] = [scope['changed_files'][0]]
                        decisions_path.write_text(json.dumps(decisions))
                        body = root / f'body-{verdict}-{omitted}.md'
                        body.write_text(packet['report'])
                        report = root / f'cli-writer-{verdict}-{omitted}.md'
                        expected = 'INCOMPLETE' if omitted else verdict
                        result = subprocess.run([
                            sys.executable, str(HERE / 'adaptive_review.py'), 'write-report',
                            '--body-file', str(body), '--project-root', str(repo), '--source-root', str(repo),
                            '--review-output-path', str(report), '--runner-mode', 'adversarial-cli',
                            '--specialists-completed', '0', '--consolidator-completed', 'true',
                            '--adversarial-model', 'codex', '--fallback-reason', 'native review unavailable',
                            '--base-commit', scope['base_commit'], '--diff-path', scope['diff_path'],
                            '--verdict', verdict, '--scope-file', str(scope_file), '--review-decisions', str(decisions_path),
                        ], capture_output=True, text=True)
                        metadata = json.loads(result.stdout)
                        self.assertEqual(metadata['verdict'], expected, result.stdout + result.stderr)
                        self.assertEqual(metadata['runner_mode'], 'incomplete' if omitted else 'adversarial-cli')
                        self.assertEqual(result.returncode, 2 if omitted else 0)
                        self.assertEqual(metadata['specialists_completed'], 0)
                        self.assertEqual(metadata['consolidator_completed'], not omitted)
                        self.assertIn(f'`consolidator_completed={not omitted}`', report.read_text())
                        if omitted:
                            self.assertIn(scope['changed_files'][1], metadata['fallback_reason'])
                            self.assertIn(metadata['fallback_reason'], report.read_text())
                        self.assertTrue(report.exists())
                        saved_report = report.read_text()
                        self.assertIn(f'**Verdict:** {expected}', saved_report)
                        self.assertIn(f'`runner_mode={metadata["runner_mode"]}`', saved_report)
                        if omitted:
                            current_body = saved_report.split("---", 1)[1]
                            self.assertIn('CURRENT_REVIEW_STATUS: INCOMPLETE', current_body)
                            self.assertIn('VERDICT: INCOMPLETE', current_body)
                            self.assertNotRegex(current_body, r'CURRENT_REVIEW_STATUS:\s*COMPLETE')
                            self.assertNotRegex(current_body, r'VERDICT\s*:?\s*PASS(?:-WITH-TRIAGE)?\b')
                        decisions_bytes = Path(metadata['retained_decisions']).read_bytes()
                        decisions_digest = hashlib.sha256(decisions_bytes).hexdigest()
                        self.assertEqual(json.loads(decisions_bytes), decisions)
                        self.assertEqual(metadata['decisions_sha256'], decisions_digest)
                        self.assertRegex(saved_report, rf'Review decisions SHA-256:\*\* `{decisions_digest}`')

    @unittest.skipIf(os.name == 'nt', 'uses shell executable fixture')
    def test_public_writers_reject_pass_with_triage_without_retained_findings(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            prompt = root / 'prompt'
            prompt.write_text('Inspect the frozen candidate and return a complete review.')
            packet = review_packet(scope)
            packet['report'] = ('CURRENT_REVIEW_STATUS: COMPLETE\n'
                                'Scope: every changed file and relevant caller was inspected. '
                                'Findings: none. The candidate satisfies the stated contract. '
                                'VERDICT: PASS-WITH-TRIAGE')
            packet['decisions']['findings'] = []
            reviewer = root / 'reviewer'
            fake_reviewer(reviewer, packet)
            run_cli_report = root / 'run-cli.md'
            run_cli = subprocess.run([
                sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                '--prompt-file', str(prompt), '--project-root', str(repo), '--source-root', str(repo),
                '--diff-path', scope['diff_path'], '--scope-file', str(scope_file),
                '--base-commit', scope['base_commit'], '--review-output-path', str(run_cli_report),
                '--host', 'claude', '--codex-command', str(reviewer), '--claude-command', str(reviewer),
                '--no-workflow-available', '--no-agent-dispatch-available',
            ], capture_output=True, text=True)
            run_cli_metadata = json.loads(run_cli.stdout)
            self.assertEqual(run_cli_metadata['verdict'], 'INCOMPLETE', run_cli.stdout + run_cli.stderr)
            self.assertEqual(run_cli.returncode, 2)
            self.assertNotIn('VERDICT: PASS-WITH-TRIAGE', run_cli_report.read_text())

            decisions = root / 'decisions.json'
            decisions.write_text(json.dumps(packet['decisions']))
            body = root / 'body.md'
            body.write_text(packet['report'])
            direct_report = root / 'write-report.md'
            direct = subprocess.run([
                sys.executable, str(HERE / 'adaptive_review.py'), 'write-report',
                '--body-file', str(body), '--project-root', str(repo), '--source-root', str(repo),
                '--review-output-path', str(direct_report), '--runner-mode', 'adversarial-cli',
                '--specialists-completed', '0', '--consolidator-completed', 'true',
                '--adversarial-model', 'codex', '--fallback-reason', 'fixture fallback',
                '--base-commit', scope['base_commit'], '--diff-path', scope['diff_path'],
                '--verdict', 'PASS-WITH-TRIAGE', '--scope-file', str(scope_file),
                '--review-decisions', str(decisions),
            ], capture_output=True, text=True)
            direct_metadata = json.loads(direct.stdout)
            self.assertEqual(direct_metadata['verdict'], 'INCOMPLETE', direct.stdout + direct.stderr)
            self.assertEqual(direct.returncode, 2)
            saved = direct_report.read_text()
            self.assertIn('**Verdict:** INCOMPLETE', saved)
            self.assertIn('VERDICT: INCOMPLETE', saved)
            self.assertNotIn('VERDICT: PASS-WITH-TRIAGE', saved)

    def test_cli_pass_with_open_blocker_is_incomplete(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            prompt = root / 'prompt.txt'
            prompt.write_text('review')
            packet = review_packet(scope)
            packet['report'] = 'REVIEWED SCOPE: all files. FINDINGS: blockers remain despite a thorough check of all callers. VERDICT: PASS'
            packet['decisions']['findings'] = [{'file': 'source', 'line': 1, 'severity': 'blocker', 'rule': 'BUG', 'summary': 'broken', 'diagnosis': 'wrong', 'disposition': 'open', 'reason': 'not refuted'}]
            fake = root / 'reviewer'
            fake_reviewer(fake, packet)
            result = subprocess.run([
                sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                '--prompt-file', str(prompt), '--project-root', str(repo), '--source-root', str(repo), '--diff-path', scope['diff_path'],
                '--scope-file', str(scope_file), '--base-commit', scope['base_commit'],
                '--review-output-path', str(root / 'report.md'), '--host', 'claude',
                '--codex-command', str(fake), '--claude-command', str(fake),
                '--no-workflow-available', '--no-agent-dispatch-available',
            ], capture_output=True, text=True)
            self.assertEqual(result.returncode, 2)
            self.assertEqual(json.loads(result.stdout)['verdict'], 'INCOMPLETE')

    def test_cli_rejects_missing_scope_and_wrong_execution_root(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            prompt = root / 'prompt.txt'
            prompt.write_text('review')
            called = root / 'reviewer-called'
            fake = root / 'reviewer'
            fake.write_text(f'#!/bin/sh\ntouch "{called}"\n')
            fake.chmod(fake.stat().st_mode | stat.S_IXUSR)
            common = [sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                      '--prompt-file', str(prompt), '--source-root', str(repo), '--diff-path', scope['diff_path'],
                      '--base-commit', scope['base_commit'], '--review-output-path', str(root / 'report.md'),
                      '--host', 'claude', '--codex-command', str(fake),
                      '--no-workflow-available', '--no-agent-dispatch-available']
            for index, extra in enumerate((['--project-root', str(repo)],
                                           ['--project-root', str(root), '--scope-file', str(scope_file)])):
                invocation = common.copy()
                invocation[invocation.index('--review-output-path') + 1] = str(root / f'report-{index}.md')
                result = subprocess.run(invocation + extra, capture_output=True, text=True)
                self.assertEqual(result.returncode, 2, result.stderr)
                self.assertEqual(json.loads(result.stdout)['verdict'], 'INCOMPLETE')
                self.assertFalse(called.exists())

    def test_cli_failures_are_incomplete(self):
        cases = {
            "empty": "#!/bin/sh\nexit 0\n",
            "malformed": "#!/bin/sh\nprintf '%s\\n' 'VERDICT: PASS'\n",
            "auth": "#!/bin/sh\nprintf '%s\\n' 'authentication failed' >&2\nexit 1\n",
            "timeout": "#!/bin/sh\nsleep 2\n",
        }
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            prompt = root / 'prompt.txt'
            prompt.write_text('review')
            for name, contents in cases.items():
                fake = root / name
                fake.write_text(contents)
                fake.chmod(fake.stat().st_mode | stat.S_IXUSR)
                report = root / f'{name}.md'
                proc = subprocess.run([
                    sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                    '--prompt-file', str(prompt), '--project-root', str(repo), '--source-root', str(repo), '--diff-path', scope['diff_path'],
                    '--scope-file', str(scope_file), '--base-commit', scope['base_commit'], '--review-output-path', str(report), '--host', 'claude',
                    '--codex-command', str(fake), '--timeout', '0.1', '--no-workflow-available', '--no-agent-dispatch-available',
                ], capture_output=True, text=True, check=False)
                self.assertEqual(proc.returncode, 2, name)
                self.assertIn('INCOMPLETE', report.read_text(), name)

    def test_cli_retries_comparable_host_cli_after_preferred_failure(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            prompt = root / 'prompt.txt'
            prompt.write_text('review')
            failed = root / 'failed-reviewer'
            failed.write_text('#!/bin/sh\nexit 1\n')
            failed.chmod(failed.stat().st_mode | stat.S_IXUSR)
            successful = root / 'codex-reviewer'
            fake_reviewer(successful, review_packet(scope))
            report = root / 'report.md'
            proc = subprocess.run([
                sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                '--prompt-file', str(prompt), '--project-root', str(repo), '--source-root', str(repo), '--diff-path', scope['diff_path'],
                '--scope-file', str(scope_file), '--base-commit', scope['base_commit'], '--review-output-path', str(report), '--host', 'codex',
                '--claude-command', str(failed), '--codex-command', str(successful), '--timeout', '2',
                '--no-workflow-available', '--no-agent-dispatch-available',
            ], capture_output=True, text=True, check=False)
            self.assertEqual(proc.returncode, 0, proc.stderr)
            metadata = json.loads(proc.stdout)
            self.assertEqual(metadata['adversarial_model'], 'codex')
            self.assertIn('using codex CLI', report.read_text())

    def test_no_automated_runner_persists_incomplete(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            prompt = root / 'prompt.txt'
            prompt.write_text('review')
            report = root / 'report.md'
            env = {**os.environ, 'PATH': '/nonexistent'}
            proc = subprocess.run([
                sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                '--prompt-file', str(prompt), '--project-root', str(repo), '--source-root', str(repo), '--diff-path', scope['diff_path'],
                '--scope-file', str(scope_file), '--base-commit', scope['base_commit'], '--review-output-path', str(report), '--host', 'unknown',
                '--no-workflow-available', '--no-agent-dispatch-available',
            ], env=env, capture_output=True, text=True, check=False)
            self.assertEqual(proc.returncode, 2)
            self.assertIn('runner_mode=incomplete', report.read_text())


class CliPerStreamByteLimitTest(unittest.TestCase):
    """B085/5301: each reviewer stream has its own inclusive 8,388,608-byte allowance."""
    LIMIT = 8388608  # oracle fixed independently of adaptive_review.MAX_REVIEW_OUTPUT_BYTES

    def _run(self, root, repo, scope, scope_file, specs, tag):
        """specs maps cli -> dict(stdout_len, stderr_len, exit, malformed)."""
        prompt = root / 'prompt'
        prompt.write_text('Inspect the frozen changes.')
        attempts = root / 'attempts'
        attempts.write_text('')

        def reviewer(cli):
            spec = specs[cli]
            packet = review_packet(scope)
            packet['report'] += '\nReview supplied by ' + cli
            if cli == 'claude':
                packet['decisions']['reviewed_files'].append('README-' + cli)
            wire = b'not json' if spec.get('malformed') else json.dumps(packet).encode()
            if spec.get('stdout_len'):
                wire = wire + b' ' * (spec['stdout_len'] - len(wire))
            path = root / cli
            path.write_text(
                '#!/usr/bin/env python3\nimport os, sys\n'
                'with open("source", "rb") as f: assert f.read() == b"new"\n'
                f'with open({str(attempts)!r}, "a") as f: print({cli!r}, file=f)\n'
                f'os.write(1, {wire!r})\n'
                f'os.write(2, bytes([233]) * {spec.get("stderr_len", 0)})\n'
                f'sys.exit({spec.get("exit", 0)})\n')
            path.chmod(0o700)
            return path

        report = root / (tag + '.md')
        result = subprocess.run([sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
            '--project-root', str(repo), '--source-root', str(repo), '--base-commit', scope['base_commit'],
            '--scope-file', str(scope_file), '--diff-path', scope['diff_path'], '--prompt-file', str(prompt),
            '--review-output-path', str(report), '--no-workflow-available', '--no-agent-dispatch-available',
            '--host', 'claude', '--requested-model', 'requested', '--effective-model', 'verified-preferred',
            '--fallback-effective-model', 'verified-second',
            '--codex-command', str(reviewer('codex')), '--claude-command', str(reviewer('claude'))],
            capture_output=True, text=True)
        return result, json.loads(result.stdout), report.read_text(), attempts.read_text().split()

    def _expect(self, run, calls, selected):
        result, metadata, saved, seen = run
        self.assertEqual(seen, calls)
        self.assertEqual(metadata['specialists_completed'], 0)
        if selected is None:
            self.assertEqual(result.returncode, 2, result.stderr)
            self.assertEqual(metadata['verdict'], 'INCOMPLETE')
            self.assertFalse(metadata['consolidator_completed'])
            self.assertIsNone(metadata['adversarial_model'])
            self.assertIsNone(metadata['effective_model'])
            self.assertNotIn('retained_decisions', metadata)
            self.assertIn('**Verdict:** INCOMPLETE', saved)
            self.assertNotIn('Review supplied by', saved)
            return
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(metadata['verdict'], 'PASS')
        self.assertTrue(metadata['consolidator_completed'])
        self.assertEqual(metadata['adversarial_model'], selected)
        self.assertEqual(metadata['requested_model'], 'requested')
        self.assertEqual(metadata['effective_model'], 'verified-preferred' if selected == 'codex' else 'verified-second')
        self.assertIn('Review supplied by ' + selected, saved)
        self.assertNotIn('Review supplied by ' + ('claude' if selected == 'codex' else 'codex'), saved)
        sidecar = Path(metadata['retained_decisions']).read_bytes()
        self.assertEqual(hashlib.sha256(sidecar).hexdigest(), metadata['decisions_sha256'])
        self.assertIn(metadata['decisions_sha256'], saved)
        # Only the selected reviewer's own declaration is retained.
        self.assertEqual('README-claude' in json.loads(sidecar)['reviewed_files'], selected == 'claude')

    @unittest.skipIf(os.name == 'nt', 'uses POSIX process groups and executable fixtures')
    def test_each_stream_has_its_own_inclusive_limit_at_cli_selection(self):
        L = self.LIMIT
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)

            def check(tag, specs, calls, selected):
                with self.subTest(case=tag):
                    self._expect(self._run(root, repo, scope, scope_file, specs, tag), calls, selected)

            ok = {}
            # 1. preferred stderr: exactly at the limit wins despite stdout+stderr exceeding it; one byte over falls back.
            check('preferred-stderr-at-limit', {'codex': {'stderr_len': L}, 'claude': ok}, ['codex'], 'codex')
            check('preferred-stderr-over', {'codex': {'stderr_len': L + 1}, 'claude': ok}, ['codex', 'claude'], 'claude')
            # 2. alternate boundary, behind each independent preferred failure.
            for name, failure in (('nonzero', {'exit': 3}), ('malformed', {'malformed': True})):
                check(name + '-alternate-stderr-at-limit', {'codex': failure, 'claude': {'stderr_len': L}},
                      ['codex', 'claude'], 'claude')
                check(name + '-alternate-stderr-over', {'codex': failure, 'claude': {'stderr_len': L + 1}},
                      ['codex', 'claude'], None)
            check('both-stderr-over', {'codex': {'stderr_len': L + 1}, 'claude': {'stderr_len': L + 1}},
                  ['codex', 'claude'], None)
            check('restored-producer', {'codex': {'stderr_len': L}, 'claude': {'stderr_len': L + 1}}, ['codex'], 'codex')
            # 3. stdout allowance is independent: legal JSON whitespace padding to exactly the limit.
            check('preferred-stdout-at-limit', {'codex': {'stdout_len': L}, 'claude': ok}, ['codex'], 'codex')
            check('preferred-stdout-over', {'codex': {'stdout_len': L + 1}, 'claude': ok}, ['codex', 'claude'], 'claude')
            check('alternate-stdout-at-limit', {'codex': {'exit': 3}, 'claude': {'stdout_len': L}},
                  ['codex', 'claude'], 'claude')
            check('alternate-stdout-over', {'codex': {'exit': 3}, 'claude': {'stdout_len': L + 1}},
                  ['codex', 'claude'], None)
            check('both-streams-at-limit', {'codex': {'stdout_len': L, 'stderr_len': L}, 'claude': ok}, ['codex'], 'codex')
            check('both-streams-at-limit-alternate', {'codex': {'exit': 3}, 'claude': {'stdout_len': L, 'stderr_len': L}},
                  ['codex', 'claude'], 'claude')


if __name__ == '__main__':
    unittest.main()


class HostCarryPlanTest(unittest.TestCase):
    """B087 B01: the Python host attaches a parent-supplied carry plan at write-report."""

    def _fixture(self, root):
        repo = root / 'repo'
        repo.mkdir()
        def git(*args, env=None):
            return subprocess.run(['git', '-c', 'commit.gpgsign=false', '-c', 'core.hooksPath=/dev/null', *args], cwd=repo, check=True, capture_output=True, text=True, env=env).stdout.strip()
        git('init', '-b', 'main')
        git('config', 'user.name', 'Test')
        git('config', 'user.email', 'test@example.invalid')
        (repo / 'leaf-a.md').write_text('a0')
        (repo / 'leaf-b.md').write_text('b0')
        git('add', '.')
        git('commit', '-m', 'base')
        git('checkout', '-b', 'candidate')
        (repo / 'leaf-a.md').write_text('a1')
        (repo / 'leaf-b.md').write_text('b1')
        git('commit', '-am', 'round one')
        predecessor = git('rev-parse', 'HEAD')
        (repo / 'leaf-b.md').write_text('b2')
        git('commit', '-am', 'round two')
        capture = subprocess.run(['bash', str(HERE / 'get_diff.sh')], cwd=repo, check=True, capture_output=True, text=True, env={**os.environ, 'REVIEW_PREDECESSOR_COMMIT': predecessor})
        scope_file = root / 'scope.json'
        scope_file.write_text(capture.stdout)
        return repo, json.loads(capture.stdout), scope_file, predecessor

    def _write(self, root, repo, scope, scope_file, angles, plan, name):
        angle_file = root / f'angles-{name}.json'
        angle_file.write_text(json.dumps(angles))
        body = root / 'body.md'
        body.write_text('Reviewed scope: all files and callers. Findings: none. Detailed verification completed. VERDICT: PASS')
        report = root / f'report-{name}.md'
        decisions = root / 'decisions.json'
        decisions.write_text(json.dumps({'candidate_identity': angles['a']['candidate_identity'], 'findings': []}))
        argv = [sys.executable, str(HERE / 'adaptive_review.py'), 'write-report', '--runner-mode', 'canonical-six-angle',
                '--specialists-completed', '6', '--consolidator-completed', 'true', '--verdict', 'PASS',
                '--angle-results', str(angle_file), '--body-file', str(body), '--review-output-path', str(report),
                '--scope-file', str(scope_file), '--review-decisions', str(decisions), '--project-root', str(repo), '--source-root', str(repo),
                '--diff-path', scope['diff_path'], '--base-commit', scope['base_commit']]
        if plan is not None:
            plan_file = root / f'plan-{name}.json'
            plan_file.write_text(json.dumps(plan))
            argv += ['--carry-plan', str(plan_file)]
        result = subprocess.run(argv, capture_output=True, text=True)
        return result, report, angle_file

    def test_write_report_credits_the_unchanged_leaf_only_with_a_parent_plan(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file, predecessor = self._fixture(root)
            self.assertEqual(scope['successor_delta']['changed_files'], ['leaf-b.md'])
            self.assertEqual(sorted(scope['changed_files']), ['leaf-a.md', 'leaf-b.md'])
            identity = {'commit': scope['candidate_commit'], 'tree': scope['candidate_tree'], 'diff_sha256': scope['diff_sha256']}
            packet = {'candidate_identity': identity, 'reviewed_files': ['leaf-b.md'], 'findings': []}
            angles = {label: packet for label in 'abcdef'}
            unit = {'version': 1, 'path': 'leaf-a.md', 'obligation': 'file_local', 'inputs': [], 'dynamic_lookup': False,
                    'predecessor_commit': predecessor, 'predecessor_run_id': 'review-a', 'predecessor_operation_digest': 'b' * 64}
            # Without the parent plan the unchanged leaf is not inspected: the writer refuses success.
            result, report, _ = self._write(root, repo, scope, scope_file, angles, None, 'none')
            self.assertEqual(result.returncode, 2, result.stderr)
            self.assertEqual(json.loads(result.stdout)['verdict'], 'INCOMPLETE')
            # With it the same reviewer replies pass, and the saved bytes carry the plan the admission reads.
            result, report, angle_file = self._write(root, repo, scope, scope_file, angles, [unit], 'plan')
            self.assertEqual(result.returncode, 0, result.stderr)
            metadata = json.loads(result.stdout)
            self.assertEqual(metadata['verdict'], 'PASS')
            self.assertEqual(metadata['specialists_completed'], 6)
            saved = json.loads(Path(metadata['retained_angles']).read_bytes())
            for label in 'abcdef':
                self.assertEqual(saved[label]['carried_units'], [unit])
                self.assertEqual(saved[label]['reviewed_files'], ['leaf-b.md'])
                self.assertIs(saved[label]['cross_file_fresh'], True)
            self.assertIn(hashlib.sha256(Path(metadata['retained_angles']).read_bytes()).hexdigest(), report.read_text())
            # A plan naming a path the successor changed, or another predecessor, is refused.
            changed = {**unit, 'path': 'leaf-b.md'}
            result, _, _ = self._write(root, repo, scope, scope_file, angles, [changed], 'changed')
            self.assertEqual(result.returncode, 2)
            result, _, _ = self._write(root, repo, scope, scope_file, angles, [{**unit, 'predecessor_commit': 'c' * 40}], 'other')
            self.assertEqual(result.returncode, 2)


class MaterialPacketWriterTest(unittest.TestCase):
    """B087 B07: the shipped writer retains both material packets and binds their digests in the header."""

    def test_write_report_binds_both_material_packets(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            reply = review_packet(scope)['decisions']
            decisions = root / 'decisions.json'
            decisions.write_text(json.dumps(reply))
            angles = root / 'angles.json'
            angles.write_text(json.dumps({label: reply for label in 'abcdef'}))
            body = root / 'body.md'
            body.write_text('Reviewed scope: all files and callers. Findings: none. Detailed verification completed.\nMATERIAL_DECISION: DEC-1\nVERDICT: PASS')
            impacts, dispositions = root / 'impacts.json', root / 'dispositions.json'
            impacts.write_bytes(b'{"decisions": []}\n')
            dispositions.write_bytes(b'{"decisions": [] }')

            def write(name, *extra):
                report = root / f'{name}.md'
                result = subprocess.run([
                    sys.executable, str(HERE / 'adaptive_review.py'), 'write-report', '--runner-mode', 'canonical-six-angle',
                    '--specialists-completed', '6', '--consolidator-completed', 'true', '--verdict', 'PASS',
                    '--angle-results', str(angles), '--review-decisions', str(decisions), '--body-file', str(body),
                    '--review-output-path', str(report), '--scope-file', str(scope_file), '--project-root', str(repo),
                    '--source-root', str(repo), '--diff-path', scope['diff_path'], '--base-commit', scope['base_commit'], *extra,
                ], capture_output=True, text=True)
                return result, report

            result, report = write('both', '--material-impacts', str(impacts), '--material-dispositions', str(dispositions))
            self.assertEqual(result.returncode, 0, result.stderr)
            metadata = json.loads(result.stdout)
            header = report.read_text().split('\n---\n')[0]
            for label, path, key in (('Expected material impacts', impacts, 'retained_material_impacts'), ('Material dispositions', dispositions, 'retained_material_dispositions')):
                retained = Path(metadata[key])
                self.assertEqual(retained.read_bytes(), path.read_bytes())
                self.assertIn(f'**{label} SHA-256:** `{hashlib.sha256(path.read_bytes()).hexdigest()}`', header)
            # One packet alone cannot support a material decision: the report is INCOMPLETE and binds nothing.
            result, report = write('one', '--material-impacts', str(impacts))
            self.assertEqual(result.returncode, 2, result.stderr)
            self.assertEqual(json.loads(result.stdout)['verdict'], 'INCOMPLETE')
            self.assertNotIn('material impacts SHA-256', report.read_text())
            # Without packets the header is unchanged from the legacy writer.
            result, report = write('none')
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertNotIn('material', report.read_text().split('\n---\n')[0].lower())


class CarriedCreditShapeTest(unittest.TestCase):
    """Shared versioned fixture; the Go admission and native workflow tests read the same file."""

    MESSAGES = {
        "cross_file": "fresh cross-file", "closed_inputs": "closed input list", "version": "credit version",
        "identity": "predecessor identity", "path": "changed since the predecessor", "input": "declared input",
        "duplicate": "duplicated", "overlap": "both fresh and carried", "missing": "incomplete reviewed_files coverage",
    }

    def test_shared_credit_cases(self):
        doc = json.loads((Path(__file__).parent / "testdata" / "credit_cases.json").read_text())
        scope = {"changed_files": doc["changed_files"], "successor_delta": doc["successor_delta"]}
        for case in doc["cases"]:
            with self.subTest(case=case["name"]):
                if case["error_key"] is None:
                    adaptive_review._check_angle_coverage(case["angle"], scope)
                else:
                    with self.assertRaisesRegex(ValueError, self.MESSAGES[case["error_key"]]):
                        adaptive_review._check_angle_coverage(case["angle"], scope)

    def test_credit_requires_a_captured_delta(self):
        doc = json.loads((Path(__file__).parent / "testdata" / "credit_cases.json").read_text())
        angle = doc["cases"][0]["angle"]
        with self.assertRaisesRegex(ValueError, "captured successor delta"):
            adaptive_review._check_angle_coverage(angle, {"changed_files": doc["changed_files"]})


class ConsolidationDiscoveryWriterTest(unittest.TestCase):
    """B087 B02: the host writer refuses a discovery without the responsible angle's fresh inspection."""

    def test_write_report_requires_a_fresh_responsible_inspection(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            reply = review_packet(scope)['decisions']
            site = {'file': 'internal/foo.go', 'start_line': 10, 'end_line': 12}
            observation = 'concrete callback is never registered when parse fails'
            inspection = {**site, 'observation': observation, 'tree': scope['candidate_tree'], 'assigned_angle': 'a', 'responder_angle': 'a'}
            body = root / 'body.md'
            body.write_text('Reviewed scope: all files and callers. Findings: none. Detailed verification completed.\nVERDICT: PASS')

            def write(name, *, packet_a, discovery):
                decisions = root / f'{name}-decisions.json'
                decisions.write_text(json.dumps({**reply, 'discoveries': discovery}))
                angles = root / f'{name}-angles.json'
                angles.write_text(json.dumps({label: (packet_a if label == 'a' else reply) for label in 'abcdef'}))
                report = root / f'{name}.md'
                result = subprocess.run([
                    sys.executable, str(HERE / 'adaptive_review.py'), 'write-report', '--runner-mode', 'canonical-six-angle',
                    '--specialists-completed', '6', '--consolidator-completed', 'true', '--verdict', 'PASS',
                    '--angle-results', str(angles), '--review-decisions', str(decisions), '--body-file', str(body),
                    '--review-output-path', str(report), '--scope-file', str(scope_file), '--project-root', str(repo),
                    '--source-root', str(repo), '--diff-path', scope['diff_path'], '--base-commit', scope['base_commit'],
                ], capture_output=True, text=True)
                return result

            discovery = [{**site, 'angle': 'a', 'observation': observation}]
            ok = write('ok', packet_a={**reply, 'site_inspections': [inspection]}, discovery=discovery)
            self.assertEqual(ok.returncode, 0, ok.stderr)
            self.assertEqual(json.loads(ok.stdout)['verdict'], 'PASS')
            for name, packet, found in (
                ('withheld', reply, discovery),
                ('neighbor', {**reply, 'site_inspections': [{**inspection, 'start_line': 20, 'end_line': 22}]}, discovery),
                ('other-tree', {**reply, 'site_inspections': [{**inspection, 'tree': 'f' * 40}]}, discovery),
                ('answered-by-e', {**reply, 'site_inspections': [{**inspection, 'responder_angle': 'e'}]}, discovery),
                ('substituted', {**reply, 'site_inspections': [inspection]}, [{**discovery[0], 'observation': 'another plausible observation'}]),
                ('removed', {**reply, 'site_inspections': [inspection]}, [{**discovery[0], 'observation': ''}]),
            ):
                with self.subTest(name):
                    result = write(name, packet_a=packet, discovery=found)
                    self.assertEqual(result.returncode, 2, result.stderr)
                    metadata = json.loads(result.stdout)
                    self.assertEqual(metadata['verdict'], 'INCOMPLETE')
                    self.assertEqual(metadata['specialists_completed'], 6)
