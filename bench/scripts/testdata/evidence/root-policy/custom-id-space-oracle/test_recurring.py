"""Committed fixture (T-E40-F06-003 round-4 code-review regression case,
finding F.1a): a mixed oracle file -- one ordinary test plus one
`@pytest.mark.parametrize(..., ids=[...])` test whose custom identifier
contains a literal space. Copied into a scratch candidate's evaluator/
directory by tc043_root_policy_isolation_test.sh so the real test-identity
collector runs real collection against it, exactly like
../parametrized-oracle/test_recurring.py's round-3 regression fixture.
Lives under bench/scripts/testdata/ (not a .sh test script) for the same
reason that one does: a real parametrize-decorated test cannot be authored
without the collection-tool tokens TC-051's AC-T3 forbidden-token sweep
looks for, and tc043 itself is one of that sweep's targets.

Before this round's fix, the old text-parsing collector's line-shape filter
(added to suppress an unrelated pytest warnings-summary false positive)
discarded ANY collect-only line containing a space in a "::"-delimited
segment -- including this real node id's custom `ids=["case with space"]`
suffix -- so this test's identity was never derived at all, and a renamed,
stripped copy of it slipped past every detection signal undetected.
"""

import pytest
from taskmanager.manager import TaskManager


def test_ordinary_recurrence_check_unaffected():
    manager = TaskManager()
    assert manager is not None


@pytest.mark.parametrize("n", [1], ids=["case with space"])
def test_add_task_accepts_recurrence_rule_custom_space_id(n):
    assert n > 0
