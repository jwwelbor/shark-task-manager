import os
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import source_inventory


@unittest.skipUnless(os.open in os.supports_dir_fd and hasattr(os, 'O_NOFOLLOW'), 'requires descriptor-relative open')
class SourceInventoryRaceTests(unittest.TestCase):
    def test_swapped_symlink_and_fifo_are_rejected(self):
        for kind in ('symlink', 'fifo'):
            with self.subTest(kind=kind), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp) / 'root'
                root.mkdir()
                path = root / 'entry'
                path.write_text('original')
                outside = Path(tmp) / 'outside'
                outside.write_text('secret')
                root_fd = os.open(root, os.O_RDONLY | os.O_DIRECTORY)
                original = os.stat
                swapped = False
                def stat_then_swap(*args, **kwargs):
                    nonlocal swapped
                    result = original(*args, **kwargs)
                    if not swapped and args[0] == b'entry':
                        swapped = True
                        path.unlink()
                        if kind == 'fifo':
                            os.mkfifo(path)
                        else:
                            path.symlink_to(outside)
                    return result
                try:
                    with mock.patch.object(source_inventory.os, 'stat', side_effect=stat_then_swap):
                        with self.assertRaises((OSError, ValueError)):
                            source_inventory._entry(root_fd, b'entry')
                finally:
                    os.close(root_fd)


if __name__ == '__main__':
    unittest.main()
