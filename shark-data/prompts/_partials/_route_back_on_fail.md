{{define "_route_back_on_fail"}}# Route entity back on FAIL

On FAIL, record a blocker note, set the bug-fix flag, and reset status to development with a rationale doc reference.

```bash
shark task note add $TASK_ID --type blocker \
  --content "{{.fail_reason_summary}} — see ${ARTIFACT_PATH}"
shark task context set $TASK_ID --field bug_fix --value true
shark status set $TASK_ID {{.fail_target_status | default "ready_for_development"}} \
  --reason="{{.fail_reason_summary}}" \
  --reason-doc="${ARTIFACT_PATH}"
```

Variables:
- `fail_reason_summary` — short string explaining what failed.
- `fail_target_status` — status to route back to (defaults to `ready_for_development`; UAT rejection routes here too; some workflows route to `ready_for_refinement_*` instead).
- `ARTIFACT_PATH` — bash variable in scope.

For multi-task FAIL routing (e.g., UAT rejecting some tasks but approving others), call this partial in a loop over the rejected task IDs.
{{end}}
