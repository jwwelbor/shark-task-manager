Bug {{.id}} is blocked. Do not spawn agent — wait for human resolution.
{{if .blocked_reason}}
Blocked reason: {{.blocked_reason}}
{{end}}
Find the step the item was parked from (the old status of its newest history row into blocked): shark status history {{.id}}
Once resolved, resume the item at that step: shark status set {{.id}} <parked step> --reason "<the decision or resolution>"
