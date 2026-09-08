"""Held-back oracle for py-epic-task-organization (REQ-F-011, AC-F11-35).

Two feature slices (task tagging; tag-filtered listing) plus one end-to-end
integration test spanning both, matching final_predicate.kind:
descendant_oracles_union's integration_test_ids + child_oracles operand
shape (the same shape as child_oracles_union, T-E40-F11-009/ADR-F11-04).
Neither Task.tags nor TaskManager.tag_task/list_tasks(tag=...) exists in the
fixture's base state (base_sha 964fa68e4c9e0c4e0f3756d9efd78b888c558fd9), so
every function here fails at base and passes only after
evaluator/reference.patch is applied. Non-regression of existing
add_task/complete/list_tasks/overdue behavior is proven by the shared,
unmodified test_manager.py suite remaining green under the absolute P2P
clause (REQ-F-017) -- the same convention py-feature-recurring-tasks uses,
not duplicated here as a dedicated function.
"""

from taskmanager.manager import TaskManager


def test_add_task_accepts_tags():
    """Child oracle 1 (feature slice: task tagging) -- add_task's own tags
    parameter, independent of tag_task."""
    manager = TaskManager()
    task = manager.add_task("Write spec", tags=["planning", "docs"])
    assert task.tags == ["planning", "docs"]


def test_tag_task_is_idempotent():
    """Child oracle 2 (feature slice: task tagging) -- tag_task() called
    directly, not through add_task, and re-tagging must not duplicate."""
    manager = TaskManager()
    task = manager.add_task("Write spec")
    manager.tag_task(task.id, "urgent")
    manager.tag_task(task.id, "urgent")
    assert manager.list_tasks()[0].tags == ["urgent"]


def test_list_tasks_filters_by_tag():
    """Child oracle 3 (feature slice: filtered listing) -- list_tasks(tag=...)
    called on manually-tagged tasks, independent of add_task/tag_task wiring."""
    manager = TaskManager()
    manager.add_task("Renew license", tags=["admin"])
    manager.add_task("Ship feature", tags=["dev"])
    manager.add_task("Write tests", tags=["dev", "admin"])

    dev_titles = [t.title for t in manager.list_tasks(tag="dev")]
    assert dev_titles == ["Ship feature", "Write tests"]


def test_list_tasks_without_filter_is_unchanged():
    """Child oracle 4 (feature slice: filtered listing) -- the no-filter
    path must remain the prior full, unfiltered listing behavior."""
    manager = TaskManager()
    manager.add_task("Renew license")
    manager.add_task("Ship feature", tags=["dev"])
    assert [t.title for t in manager.list_tasks()] == ["Renew license", "Ship feature"]


def test_tagging_and_filtered_listing_integrate_end_to_end():
    """Integration test (spans both feature slices): tag via add_task and
    via tag_task, then confirm the filtered listing reflects both paths and
    excludes untagged/differently-tagged tasks -- proving the two feature
    slices (tagging, filtered listing) are wired together, not merely
    independently correct."""
    manager = TaskManager()
    seeded = manager.add_task("Write spec", tags=["planning"])
    manager.tag_task(seeded.id, "urgent")
    other = manager.add_task("Unrelated task")
    manager.tag_task(other.id, "planning")
    manager.add_task("Different topic", tags=["dev"])

    planning_titles = sorted(t.title for t in manager.list_tasks(tag="planning"))
    assert planning_titles == ["Unrelated task", "Write spec"]

    urgent_titles = [t.title for t in manager.list_tasks(tag="urgent")]
    assert urgent_titles == ["Write spec"]
