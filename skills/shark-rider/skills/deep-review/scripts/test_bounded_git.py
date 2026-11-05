import os
from pathlib import Path
import subprocess
import tempfile
import time
import unittest
from unittest import mock

import source_inventory


@unittest.skipIf(os.name == 'nt', 'Unix process groups required')
class BoundedGitTests(unittest.TestCase):
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
