# Bug: tasks due today aren't listed as overdue

Our task manager is supposed to flag a task as overdue once its due date has
arrived. Right now, a task due *today* doesn't show up in the overdue list
at all -- it only appears starting the day after its due date.

For example:

```python
from datetime import date
from taskmanager.manager import TaskManager

manager = TaskManager()
manager.add_task("Renew license", due_date="2026-03-15")

overdue = manager.overdue_tasks(today=date(2026, 3, 15))  # want: [task], got: []
```

Tasks due in the past are correctly reported as overdue, and tasks due in
the future are correctly excluded; only the exact-due-date case is wrong.

Fix the overdue check so that a task due today is included in
`overdue_tasks`, without changing the behavior for any other due date.
