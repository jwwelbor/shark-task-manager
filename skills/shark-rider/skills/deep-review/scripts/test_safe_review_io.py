import os
from pathlib import Path
import tempfile
import unittest

from safe_review_io import read_review_file


@unittest.skipUnless(hasattr(os, 'mkfifo') and hasattr(os, 'O_NOFOLLOW'), 'requires Unix no-follow open')
class SafeReviewIOTests(unittest.TestCase):
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
