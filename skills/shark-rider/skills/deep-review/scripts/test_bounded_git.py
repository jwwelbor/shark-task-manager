import os
import signal
import sys
from pathlib import Path
import subprocess
import tempfile
import time
import unittest
from unittest import mock

import source_inventory
from bounded_git import OutputLimitExceeded, run


@unittest.skipIf(os.name == 'nt', 'Unix process groups required')
class BoundedGitTests(unittest.TestCase):
    def test_output_limit_stops_noisy_command(self):
        for stream in ('stdout', 'stderr'):
            with self.subTest(stream=stream), self.assertRaises(OutputLimitExceeded):
                run([sys.executable, '-c', f"import sys; sys.{stream}.buffer.write(b'x' * 1000000)"], capture_output=True, max_output_bytes=1024, timeout=5)

    def test_bounded_capture_preserves_text_and_errors(self):
        result = run([sys.executable, '-c', "import sys; print('ok'); print('problem', file=sys.stderr)"], capture_output=True, text=True, max_output_bytes=1024)
        self.assertEqual(result.stdout, 'ok\n')
        self.assertEqual(result.stderr, 'problem\n')

    def test_detached_filter_cannot_hold_captured_output_open(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            def git(*args):
                subprocess.run(['git', '-C', tmp, '-c', 'commit.gpgsign=false', '-c', 'core.hooksPath=/dev/null', *args], check=True, capture_output=True)
            git('init', '-b', 'main')
            git('config', 'user.name', 'Test')
            git('config', 'user.email', 'test@example.invalid')
            (root / 'source').write_text('base')
            git('add', 'source')
            git('commit', '-m', 'base')
            (root / 'filter.py').write_text("import subprocess,sys,time\nchild=subprocess.Popen([sys.executable,'-c','import time; time.sleep(10)'], start_new_session=True)\nopen('detached.pid','w').write(str(child.pid))\ntime.sleep(10)\n")
            git('config', 'filter.slow.clean', f'{sys.executable} filter.py')
            (root / '.gitattributes').write_text('source filter=slow\n')
            (root / 'source').write_text('edit')
            started = time.monotonic()
            try:
                with self.assertRaises(subprocess.TimeoutExpired):
                    run(['git', '-C', tmp, 'status', '--porcelain'], timeout=0.4, capture_output=True)
                self.assertLess(time.monotonic() - started, 2)
                self.assertTrue((root / 'detached.pid').exists())
            finally:
                if (root / 'detached.pid').exists():
                    try:
                        os.kill(int((root / 'detached.pid').read_text()), signal.SIGKILL)
                    except ProcessLookupError:
                        pass

    def test_forced_shark_inventory_respects_deadline(self):
        with tempfile.TemporaryDirectory() as tmp:
            executable = Path(tmp) / 'shark'
            executable.write_text('#!/bin/sh\nsleep 3\n')
            executable.chmod(0o700)
            with mock.patch.dict(os.environ, {'SHARK_REVIEW_INVENTORY_FORCE': '1', 'SHARK_REVIEW_INVENTORY_BIN': str(executable), 'SHARK_REVIEW_GIT_TIMEOUT': '0.2'}):
                with self.assertRaises(subprocess.TimeoutExpired):
                    source_inventory.source_inventory(Path(tmp))

    def test_inventory_timeout_retires_clean_filter(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            def git(*args):
                subprocess.run(['git', '-C', tmp, '-c', 'commit.gpgsign=false', '-c', 'core.hooksPath=/dev/null', *args], check=True, capture_output=True)
            git('init', '-b', 'main')
            git('config', 'user.name', 'Test')
            git('config', 'user.email', 'test@example.invalid')
            (root / 'source').write_text('base')
            git('add', 'source')
            git('commit', '-m', 'base')
            (root / 'filter.sh').write_text('#!/bin/sh\nprintf started > started\nsleep 2\nprintf leaked > leaked\n')
            (root / 'filter.sh').chmod(0o700)
            git('config', 'filter.slow.clean', './filter.sh')
            (root / '.gitattributes').write_text('source filter=slow\n')
            (root / 'source').write_text('edit')
            with mock.patch.dict(os.environ, {'SHARK_REVIEW_GIT_TIMEOUT': '0.5'}):
                with self.assertRaises(subprocess.TimeoutExpired):
                    source_inventory.source_inventory(root)
            self.assertTrue((root / 'started').exists())
            time.sleep(2.2)
            self.assertFalse((root / 'leaked').exists(), 'filter survived inventory timeout')
