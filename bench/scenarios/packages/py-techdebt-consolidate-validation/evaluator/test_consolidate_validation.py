"""Held-back regression test for py-techdebt-consolidate-validation
(evaluator-only). Not part of bench/fixture-py's committed suite:
adapter.sh's inject-tests capability places this file under the checkout's
tests/ directory only when a run actually needs it (REQ-F-009).

Asserts the task-field validation duplicated inline across
taskmanager/bulk_import.py, taskmanager/cli.py, and taskmanager/reminders.py
behaves identically to taskmanager/validation.non_empty_title, both before
and after the reference patch consolidates the three inline copies into a
shared call -- proving the refactor is behavior-preserving (spec.md
p2p_plus_rule_drop row: "behavior preserved"), not merely lint-clean. The
final_predicate itself carries no test id operands for this kind (REQ-F-010
Final predicate vocabulary table); this oracle is admitted as a normal
member of final_predicate.p2p_selection's absolute "every entry pass"
clause, same as every other test the include resolves to.
"""

import pytest

from taskmanager.bulk_import import import_titles
from taskmanager.cli import parse_title_arg
from taskmanager.reminders import format_reminder


def test_import_titles_strips_and_rejects_blank():
    assert import_titles(["  Buy milk  ", "Ship feature"]) == ["Buy milk", "Ship feature"]
    with pytest.raises(ValueError):
        import_titles(["   "])


def test_parse_title_arg_strips_and_rejects_blank():
    assert parse_title_arg("  Renew license  ") == "Renew license"
    with pytest.raises(ValueError):
        parse_title_arg("")


def test_format_reminder_strips_and_rejects_blank():
    assert format_reminder("  Write spec  ") == "Reminder: Write spec"
    with pytest.raises(ValueError):
        format_reminder("")
