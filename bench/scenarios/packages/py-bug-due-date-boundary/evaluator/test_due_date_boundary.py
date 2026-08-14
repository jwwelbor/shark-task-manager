"""Held-back repro test for py-bug-due-date-boundary (evaluator-only).

Not part of bench/fixture-py's committed suite: adapter.sh's inject-tests
capability places this file under the checkout's tests/ directory only when
a run actually needs it (REQ-F-009). Asserts a task due exactly today is
reported overdue -- the boundary the fixture's base `is_overdue` gets wrong
by comparing strictly `<` instead of `<=`.
"""

from datetime import date

from taskmanager.due_date import is_overdue


def test_is_overdue_true_for_task_due_today():
    today = date(2026, 3, 15)
    assert is_overdue(today.isoformat(), today) is True
