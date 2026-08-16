"""Committed fixture (T-E40-F06-003 round-4 code-review regression case,
finding F.1b): a mixed oracle file -- one ordinary test plus one
`@pytest.mark.parametrize(..., ids=[...])` test whose custom identifier
contains the literal substring "::". Copied into a scratch candidate's
evaluator/ directory by tc043_root_policy_isolation_test.sh so the real
test-identity collector runs real collection against it, exactly like
../parametrized-oracle/test_recurring.py's round-3 regression fixture.
Lives under bench/scripts/testdata/ for the same reason that one does: a
real parametrize-decorated test cannot be authored without the
collection-tool tokens TC-051's AC-T3 forbidden-token sweep looks for, and
tc043 itself is one of that sweep's targets.

Before this round's fix, the old text-parsing collector split each
collect-only line on "::" to find the test name; this custom id's OWN
literal "::" content caused that split to take the wrong segment as the
name, emitting a garbage identity instead of the real one -- so a renamed,
stripped copy of this test also slipped past every detection signal
undetected.
"""

import pytest
from taskmanager.manager import TaskManager


def test_ordinary_recurrence_check_unaffected():
    manager = TaskManager()
    assert manager is not None


@pytest.mark.parametrize("n", [1], ids=["case::with::colons"])
def test_add_task_accepts_recurrence_rule_custom_colon_id(n):
    assert n > 0
