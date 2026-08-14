"""Held-back oracle and integration tests for py-feature-recurring-tasks
(evaluator-only). Not part of bench/fixture-py's committed suite: adapter.sh's
inject-tests capability places this file under the checkout's tests/
directory only when a run actually needs it (REQ-F-009), mirroring
py-bug-due-date-boundary's test_due_date_boundary.py.

Child oracles cover each declared sub-capability independently of
TaskManager.complete()'s internal wiring:
  - test_add_task_accepts_recurrence_rule: the recurrence-rule field itself.
  - test_generate_next_occurrence_advances_by_rule_interval: next-occurrence
    generation, called directly on TaskManager.generate_next_occurrence()
    rather than through complete().
  - test_upcoming_occurrences_filters_to_window_and_excludes_done:
    upcoming-occurrences listing, exercised on manually-constructed tasks
    rather than through generate_next_occurrence().

The integration test drives the full path through TaskManager.complete()
and asserts only loose end-to-end conditions (presence in the upcoming
window, not the exact due-date offset), so:
  - a precision bug in the recurrence interval fails its own child oracle
    (exact date assertion) without also failing the integration test (whose
    window is wide enough to tolerate a small offset), and
  - a complete()-to-generate_next_occurrence wiring bug fails the
    integration test without touching any child oracle, since each child
    oracle calls its own capability directly.
These are the two negative TC-036 states this seed's
final_predicate.kind: child_oracles_union must distinguish (test-plan.md AC
test matrix, TC-036 row, child_oracles_union states).
"""

from datetime import date

from taskmanager.manager import TaskManager


def test_add_task_accepts_recurrence_rule():
    manager = TaskManager()
    task = manager.add_task("Water plants", due_date="2026-03-01", recurrence_rule="weekly")
    assert task.recurrence_rule == "weekly"


def test_generate_next_occurrence_advances_by_rule_interval():
    manager = TaskManager()
    task = manager.add_task("Water plants", due_date="2026-03-01", recurrence_rule="weekly")

    next_task = manager.generate_next_occurrence(task)

    assert next_task.due_date == "2026-03-08"
    assert next_task.recurrence_rule == "weekly"
    assert next_task.title == "Water plants"


def test_upcoming_occurrences_filters_to_window_and_excludes_done():
    manager = TaskManager()
    in_window = manager.add_task("Renew badge", due_date="2026-03-03")
    manager.add_task("Far future", due_date="2026-04-01")
    done_in_window = manager.add_task("Already done", due_date="2026-03-04")
    manager.complete(done_in_window.id)

    upcoming = manager.upcoming_occurrences(today=date(2026, 3, 1), within_days=7)

    assert [t.id for t in upcoming] == [in_window.id]


def test_completing_recurring_task_schedules_next_occurrence_into_upcoming_list():
    manager = TaskManager()
    task = manager.add_task("Water plants", due_date="2026-03-01", recurrence_rule="weekly")

    manager.complete(task.id)

    upcoming = manager.upcoming_occurrences(today=date(2026, 3, 1), within_days=14)
    titles = [t.title for t in upcoming]
    assert titles == ["Water plants"]
    assert upcoming[0].recurrence_rule == "weekly"
    assert upcoming[0].id != task.id
