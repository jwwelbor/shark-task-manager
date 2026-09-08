"""Held-back oracle for py-task-delete-task (REQ-F-010, AC-F11-30).

Three named test functions, gated on terminal completion per AC-F11-30:
successful deletion, missing-ID behavior, and non-regression of existing
manager behavior alongside the new delete_task capability. taskmanager.manager
has no delete_task method at all in the fixture's base state (base_sha
964fa68e4c9e0c4e0f3756d9efd78b888c558fd9), so every function here fails at
base and passes only after evaluator/reference.patch is applied.
"""

from datetime import date

import pytest

from taskmanager.manager import TaskManager


def test_delete_task_removes_task():
    manager = TaskManager()
    task = manager.add_task("Write spec")
    manager.delete_task(task.id)
    assert manager.list_tasks() == []


def test_delete_task_raises_for_missing_id():
    manager = TaskManager()
    manager.add_task("Write spec")

    from taskmanager.manager import TaskNotFoundError

    with pytest.raises(TaskNotFoundError):
        manager.delete_task(999)


def test_delete_task_does_not_regress_existing_manager_behavior():
    """Non-regression arm (AC-F11-30, spec.md test-plan.md row 30): deleting
    one task must leave add/complete/list/overdue behavior for the remaining
    tasks exactly as before the delete feature landed."""
    manager = TaskManager()
    keep = manager.add_task("Renew license", due_date="2020-01-01")
    doomed = manager.add_task("Ship feature")

    manager.delete_task(doomed.id)

    assert [t.id for t in manager.list_tasks()] == [keep.id]

    manager.complete(keep.id)
    assert manager.list_tasks()[0].is_done()
    assert manager.overdue_tasks(today=date(2026, 1, 1)) == []
