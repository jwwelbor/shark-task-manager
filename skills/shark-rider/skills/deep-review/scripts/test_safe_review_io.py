import os
from pathlib import Path
import tempfile
from unittest import mock
import unittest

from safe_review_io import read_review_file


@unittest.skipUnless(hasattr(os, 'mkfifo') and hasattr(os, 'O_NOFOLLOW'), 'requires Unix no-follow open')
class SafeReviewIOTests(unittest.TestCase):
    def test_fallback_obeys_requested_output_limit(self):
        with tempfile.TemporaryDirectory() as tmp:
            executable = Path(tmp) / 'shark'
            executable.write_text('#!/bin/sh\nprintf 123456789\n')
            executable.chmod(0o700)
            with mock.patch.dict(os.environ, {'SHARK_REVIEW_IO_FORCE': '1', 'SHARK_REVIEW_INVENTORY_BIN': str(executable)}):
                with self.assertRaisesRegex(ValueError, 'output exceeds'):
                    read_review_file(Path(tmp) / 'artifact', 4)

    def test_fallback_timeout_reports_timeout(self):
        with tempfile.TemporaryDirectory() as tmp:
            executable = Path(tmp) / 'shark'
            executable.write_text('#!/bin/sh\nsleep 3\n')
            executable.chmod(0o700)
            with mock.patch.dict(os.environ, {'SHARK_REVIEW_IO_FORCE': '1', 'SHARK_REVIEW_INVENTORY_BIN': str(executable), 'SHARK_REVIEW_GIT_TIMEOUT': '0.2'}):
                with self.assertRaisesRegex(ValueError, 'timed out'):
                    read_review_file(Path(tmp) / 'artifact', 32)

    def test_fifo_symlink_and_oversized_artifact_are_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            fifo = root / 'fifo'
            os.mkfifo(fifo)
            with self.assertRaises((OSError, ValueError)):
                read_review_file(fifo, 32)
            regular = root / 'regular'
            regular.write_bytes(b'large file')
            with self.assertRaises(ValueError):
                read_review_file(regular, 2)
            link = root / 'link'
            link.symlink_to(regular)
            with self.assertRaises((OSError, ValueError)):
                read_review_file(link, 32)
            self.assertEqual(read_review_file(regular, 32), b'large file')
