Feature {{.id}} ("{{.title}}") is ACTIVE — tasks are in progress.

## Steps

1. List all tasks: {{template "list_json" .}}
2. For each task (in dependency/execution order):
   - **completed / cancelled** -> skip
   - **development** -> run `/run {task-key}` to resume
   - **draft** -> run `/run {task-key}` to start
   - **blocked / on_hold** -> skip, try next
3. When ALL tasks are completed or cancelled: {{template "advance" .}}
   (This advances the feature to the code_review verification gate.)
4. If ALL remaining tasks are blocked: report to user and STOP

Respect task dependencies — do not start a task whose dependencies are incomplete.
