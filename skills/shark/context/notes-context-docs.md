# Notes, Context & Related Documents

## Notes

Typed annotations on any entity. Used for tracking decisions, blockers, solutions, etc.

### Adding Notes

```bash
# Unified create (preferred — auto-detects entity type from key)
shark create note E01-F02-001 "Chose JWT over sessions" --type=decision
shark create note E01-F02-001 "Waiting on API spec" --type=blocker --created-by=alice
shark create note E01-F02 "Split into 3 phases" --type=decision
shark create note E01 "Phase 1 complete" --type=comment
shark create note B001 "Reproduced on Safari 17.2"
shark create note CC-001 "Approved by security team" --type=comment

```

### Note Types

`comment`, `decision`, `blocker`, `solution`, `reference`, `implementation`, `testing`, `future`, `question`, `rejection`, `requirement`, `review`

`requirement` records an amended/added requirement (used by `/shark amend`).
`review` records a code-review outcome (PASS/FAIL).

### Viewing Notes

```bash
shark task notes E01-F02-001               # All notes for task
shark task notes E01-F02-001 --type decision  # Filter by type
```

### Searching Notes

```bash
shark notes search "authentication"            # Search all notes
shark notes search "JWT" --epic=E01            # Within epic
shark notes search "bug" --type decision,solution  # By note type
shark notes search "perf" --since 2026-01-01   # Date filtered
```

## Context (Structured Resume Data)

Persistent structured fields for resuming work across sessions. Auto-detects entity type from key.

### Setting Context

```bash
# Plain string
shark context set E01-F02-001 --field current_step --value "Implementing API endpoint"

# JSON arrays
shark context set E01-F02-001 --field completed_steps --value '["Wrote tests","Set up DB"]'
shark context set E01-F02-001 --field remaining_steps --value '["Integration tests","Docs"]'
shark context set E01-F02-001 --field open_questions --value '["Should we use OAuth?"]'

# JSON object
shark context set E01-F02-001 --field implementation_decisions --value '{"auth":"JWT","db":"postgres"}'

# JSON array of objects
shark context set E01-F02-001 --field blockers --value '[{"description":"Missing API key"}]'
shark context set E01-F02-001 --field acceptance_criteria_status --value '[{"criterion":"Login works","met":true}]'
```

### Available Fields

| Field | Type | Description |
|-------|------|-------------|
| `current_step` | String | What you're working on now |
| `completed_steps` | JSON array | Steps finished |
| `remaining_steps` | JSON array | Steps left to do |
| `implementation_decisions` | JSON object | Key decisions made |
| `open_questions` | JSON array | Unresolved questions |
| `blockers` | JSON array of objects | Current blockers |
| `acceptance_criteria_status` | JSON array of objects | AC tracking |

### Getting Context

```bash
shark context E01-F02-001                  # Human-readable display
shark context E01-F02-001 --json           # JSON output
shark context E01-F02                      # Feature context
shark context E01                          # Epic context
```

### Clearing Context

```bash
shark context clear E01-F02-001            # Remove all context data
```

## Related Documents

Link supporting documents (specs, designs, research) to any entity. Alias: `shark docs`.

### Adding Documents

```bash
shark related-docs add "Feature PRD" docs/plan/E01/E01-F02/feature.md --feature=E01-F02
shark related-docs add "API Design" docs/design/api.md --epic=E01
shark related-docs add "Test Plan" docs/testing/plan.md --task=E01-F02-001
shark related-docs add "Root Cause Analysis" docs/rca.md --bug=B001
shark related-docs add "Migration Plan" docs/migrate.md --change=CC-001
```

Exactly one of `--epic`, `--feature`, `--task`, `--bug`, or `--change` is required.

If the document already exists (same title+path), it is reused and linked. Linking is idempotent.

### Listing Documents

```bash
shark related-docs list --feature=E01-F02
shark related-docs list --epic=E01
shark related-docs list --task=E01-F02-001 --json
shark related-docs list --bug=B001
shark related-docs list --change=CC-001
```

### Removing Document Links

```bash
shark related-docs delete "Feature PRD" --feature=E01-F02
shark related-docs delete "Root Cause Analysis" --bug=B001
```

Delete is idempotent — succeeds even if the document is not currently linked to the parent. The document record itself is not deleted, only the link.

## Search (File-Based)

Find tasks by files they touched during completion:

```bash
shark search --file="useTheme.ts"
shark search --file="task_repository" --epic E01
```
