{{define "_advance"}}# Advance entity status with rationale

On PASS / approve, record the verdict as a typed note pointing at any artifact the workflow produced, then advance to the next status.

```bash
shark task note add $TASK_ID --type {{.advance_note_type | default "review"}} \
  --content "{{.advance_summary}} — see ${ARTIFACT_PATH}"
shark status advance $TASK_ID
```

Variables:
- `advance_note_type` — note type to use (defaults to `review`; common alternatives: `testing`, `decision`, `approval`).
- `advance_summary` — short summary string the host supplies.
- `ARTIFACT_PATH` — bash variable in scope (path to the artifact this workflow produced).
{{end}}
