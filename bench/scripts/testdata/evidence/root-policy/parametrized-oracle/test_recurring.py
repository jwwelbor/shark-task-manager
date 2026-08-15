"""Committed fixture (T-E40-F06-003 round-3 code-review regression case,
finding F.2): a parametrized oracle test, copied into a scratch candidate's
evaluator/ directory by tc043_root_policy_isolation_test.sh so the real
test-identity collector runs real collection against it. Lives under
bench/scripts/testdata/ (not a .sh test script) specifically so this file's
own real parametrize markup never appears as literal text inside
tc043_root_policy_isolation_test.sh itself -- tc043 is one of TC-051's
AC-T3 forbidden-token sweep targets, and a real parametrized pytest test
cannot be authored without the tokens that sweep looks for.
"""

import pytest


@pytest.mark.parametrize("n", [1, 2, 3])
def test_add_task_accepts_recurrence_rule_param(n):
    assert n > 0
