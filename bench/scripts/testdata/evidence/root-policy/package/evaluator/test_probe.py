"""Committed fixture stand-in for a held-back oracle test file (REQ-F-010
evaluator_only.oracle_tests[0]). Never collected by pytest as part of this
repository's own suite -- it lives under bench/scripts/testdata/, outside
any Python package pytest discovers -- and is consumed only for its
path/digest/stem identity by verify-evidence-roots.sh's offline test paths.
"""


def test_probe_always_passes():
    assert 1 + 1 == 2
