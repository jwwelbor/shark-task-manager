import os
import ctypes
import importlib
import types
import signal
import sys
from pathlib import Path
import subprocess
import tempfile
import time
import threading
import unittest
from unittest import mock

import source_inventory
from bounded_git import OutputLimitExceeded, run


class LegacyWindowsPipeTests(unittest.TestCase):
    def test_win32_pipe_result_mapping(self):
        class Function:
            def __init__(self, fn):
                self.fn = fn
            def __call__(self, *args):
                return self.fn(*args)
        state = {'error': 0, 'data': b'abc'}
        def read(handle, buffer, size, count, overlapped):
            if state['error']:
                return 0
            ctypes.memmove(buffer, state['data'], len(state['data']))
            count._obj.value = len(state['data'])
            return 1
        kernel = types.SimpleNamespace(SetNamedPipeHandleState=Function(lambda *args: 1), ReadFile=Function(read))
        fake_msvcrt = types.SimpleNamespace(get_osfhandle=lambda fd: fd)
        with mock.patch.dict(sys.modules, {'msvcrt': fake_msvcrt}), \
             mock.patch.object(ctypes, 'WinDLL', create=True, return_value=kernel), \
             mock.patch.object(ctypes, 'get_last_error', create=True, side_effect=lambda: state['error']):
            sys.modules.pop('windows_pipe', None)
            try:
                pipe = importlib.import_module('windows_pipe').WindowsPipe()
                pipe.set_nonblocking(4)
                self.assertEqual(pipe.read(4, 16), b'abc')
                state['error'] = 232
                with self.assertRaises(BlockingIOError):
                    pipe.read(4, 16)
                state['error'] = 109
                self.assertEqual(pipe.read(4, 16), b'')
            finally:
                sys.modules.pop('windows_pipe', None)


class PortableCaptureTests(unittest.TestCase):
    def test_output_limit_stops_noisy_command(self):
        for stream in ('stdout', 'stderr'):
            with self.subTest(stream=stream), self.assertRaises(OutputLimitExceeded):
                run([sys.executable, '-c', f"import sys; sys.{stream}.buffer.write(b'x' * 1000000)"], capture_output=True, max_output_bytes=1024, timeout=5)

    def test_bounded_capture_preserves_text_and_errors(self):
        result = run([sys.executable, '-c', "import sys; print('ok'); print('problem', file=sys.stderr)"], capture_output=True, text=True, max_output_bytes=1024)
        self.assertEqual(result.stdout, 'ok\n')
        self.assertEqual(result.stderr, 'problem\n')


@unittest.skipIf(os.name == 'nt', 'Unix process groups required')
class BoundedGitTests(unittest.TestCase):
    def test_capture_read_failure_reaches_inventory_caller(self):
        with tempfile.TemporaryDirectory() as tmp:
            subprocess.run(['git', '-C', tmp, 'init', '-b', 'main'], check=True, capture_output=True)
            real_read = os.read
            (Path(tmp) / 'dirty').write_text('untracked')
            for failure in (OSError, ValueError):
                def fail_reader(fd, size):
                    if threading.current_thread().name == 'bounded-git-reader':
                        raise failure('injected capture read error')
                    return real_read(fd, size)
                with self.subTest(failure=failure), mock.patch('bounded_git.os.read', side_effect=fail_reader):
                    with self.assertRaisesRegex(OSError, 'failed to read captured'):
                        run(['git', '-C', tmp, 'status', '--porcelain'], check=True, capture_output=True)
                    with self.assertRaisesRegex(OSError, 'failed to read captured'):
                        source_inventory.source_inventory(Path(tmp))

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
            for attempt in range(3):
                with self.subTest(attempt=attempt):
                    started = time.monotonic()
                    try:
                        with self.assertRaises(subprocess.TimeoutExpired):
                            run(['git', '-C', tmp, 'status', '--porcelain'], timeout=0.4, capture_output=True)
                        self.assertLess(time.monotonic() - started, 2)
                        self.assertTrue((root / 'detached.pid').exists())
                        self.assertFalse(any(t.name == 'bounded-git-reader' for t in threading.enumerate()), 'capture reader survived timeout')
                    finally:
                        if (root / 'detached.pid').exists():
                            try:
                                os.kill(int((root / 'detached.pid').read_text()), signal.SIGKILL)
                            except ProcessLookupError:
                                pass
                            (root / 'detached.pid').unlink()

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
