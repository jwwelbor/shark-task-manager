"""Held-back acceptance tests for py-change-priority-scale (evaluator-only).

Not part of bench/fixture-py's committed suite: adapter.sh's inject-tests
capability places this file under the checkout's tests/ directory only when
a run actually needs it (REQ-F-009). Covers the three acceptance-test cases
named in package.yaml's final_predicate.acceptance_test_ids:

  1. the new 1-5 integer scale itself (validate_priority/label_for over the
     full range);
  2. conversion of each of the three legacy string levels on already-stored
     records (taskmanager.legacy_records.EXISTING_TASK_RECORDS), not just
     newly created tasks;
  3. rejection of out-of-range integer priorities, alongside acceptance of
     an in-range one -- so a validator that still only understands the old
     string levels cannot pass this case by accident.
"""

from taskmanager.legacy_records import EXISTING_TASK_RECORDS
from taskmanager.manager import TaskManager
from taskmanager.priority import label_for, validate_priority


def test_new_scale_labels_and_range():
    assert label_for(1) == "critical"
    assert label_for(3) == "medium"
    assert label_for(5) == "trivial"


def test_legacy_records_converted_to_new_scale():
    manager = TaskManager()
    imported = manager.import_legacy_records(EXISTING_TASK_RECORDS)
    by_title = {task.title: task for task in imported}

    assert by_title["Renew TLS certificate"].priority == 2  # legacy "high"
    assert by_title["Update onboarding docs"].priority == 3  # legacy "medium"
    assert by_title["Archive Q1 reports"].priority == 4  # legacy "low"
    assert by_title["Archive Q1 reports"].is_done() is True


def test_validate_priority_rejects_out_of_range():
    validate_priority(3)  # a valid new-scale value must not raise

    for bad in (0, 6):
        try:
            validate_priority(bad)
        except ValueError:
            continue
        raise AssertionError(f"validate_priority({bad}) should have raised ValueError")
