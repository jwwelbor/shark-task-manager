# Add the ability to delete a task

`TaskManager` supports adding, completing, listing, and checking tasks for
being overdue, but there is no way to remove a task once it has been added.

Add a `delete_task(task_id)` method to `TaskManager` that permanently removes
the task with the given id.

```python
from taskmanager.manager import TaskManager

manager = TaskManager()
task = manager.add_task("Write spec")
manager.delete_task(task.id)

manager.list_tasks()  # want: []
```

If `task_id` does not name an existing task, `delete_task` must raise a clear,
specific error rather than an unrelated or generic one -- callers need to be
able to tell "task not found" apart from any other failure.

Deleting a task must not change the behavior of `add_task`, `complete`,
`list_tasks`, or `overdue_tasks` for any task that is not deleted.
