import unittest

from review_status import filter_status, runtime_prefix, selection_overlaps_runtime


class ReviewStatusTest(unittest.TestCase):
    def test_filters_only_untracked_designated_runtime(self):
        prefix = runtime_prefix('nested')
        status = (b'?? nested/.shark/runs/run/result.json\0'
                  b' M nested/.shark/runs/tracked\0'
                  b'?? .shark/runs/other-project\0'
                  b'R  renamed\0?? nested/.shark/runs/old-name\0')
        expected = (b' M nested/.shark/runs/tracked\0'
                    b'?? .shark/runs/other-project\0'
                    b'R  renamed\0?? nested/.shark/runs/old-name\0')
        self.assertEqual(filter_status(status, prefix), expected)
        with self.assertRaises(ValueError):
            filter_status(b'?? truncated', prefix)

    def test_snapshot_selection_cannot_include_runtime_directory(self):
        prefix = runtime_prefix('nested')
        for path in (b'nested', b'nested/.shark', b'nested/.shark/runs', b'nested/.shark/runs/run/file'):
            self.assertTrue(selection_overlaps_runtime(path, prefix))
        self.assertFalse(selection_overlaps_runtime(b'nested/app.go', prefix))
        self.assertFalse(selection_overlaps_runtime(b'other/.shark/runs/run/file', prefix))
        for path in ('../elsewhere', 'nested/../other', 'nested//child', 'nested\\other', ''):
            with self.assertRaises(ValueError):
                runtime_prefix(path)
