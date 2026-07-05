# /shark workflow - Inspect workflow status flow

Show the configured workflow for an entity type, or inspect a concrete entity's
current status and then show the matching workflow.

Usage:

```
/shark workflow                       # compact view for every workflow level
/shark workflow task                  # compact task workflow
/shark workflow change-card --all     # expanded change workflow
/shark workflow CC-042                # current status, transitions, change workflow
/shark workflow E07-F01-001 --json    # machine-readable status + workflow reads
```

This is a read-only wrapper around Shark CLI inspection commands.

## Step 0 - Parse args

Accept at most one non-flag target plus passthrough flags:

- `--all` passes to `shark admin workflow list` for the expanded metadata view.
- `--json` uses JSON forms for commands that support them.

If there is no target, run:

```bash
shark admin workflow list [--all] [--json]
```

## Step 1 - Decide whether target is an entity type or key

Entity-type targets:

| Input aliases | Workflow level |
|---|---|
| `epic`, `epics` | `epic` |
| `feature`, `features` | `feature` |
| `task`, `tasks` | `task` |
| `sprint`, `sprints` | `sprint` |
| `bug`, `bugs` | `bug` |
| `change`, `changes`, `change-card`, `change_card`, `change-cards`, `change_cards` | `change` |
| `tech-debt`, `tech_debt`, `tech-debts`, `tech_debts`, `td` | `tech_debt` |

For an entity-type target, run:

```bash
shark admin workflow list <workflow-level> [--all] [--json]
```

Concrete entity-key targets:

| Key pattern | Workflow level |
|---|---|
| `E##` | `epic` |
| `E##-F##` or `F##` | `feature` |
| `E##-F##-###` or `T-E##-F##-###` | `task` |
| `B###` | `bug` |
| `CC-###` or `C###` | `change` |
| `TD-###` | `tech_debt` |

For a concrete key, first read the current entity status:

```bash
shark get <key> --field status
```

Then read valid transitions:

```bash
shark status transitions <key>
```

Then show the matching workflow:

```bash
shark admin workflow list <workflow-level> [--all]
```

If the user requested `--json`, use:

```bash
shark get <key> --json
shark status transitions <key> --json
shark admin workflow list <workflow-level> --json
```

Summarize the status and transitions before the workflow output in human mode.

## Step 2 - Invalid target handling

If the target is neither a supported entity type nor a valid key shape, report:

```
Unsupported workflow target "<target>".

Use an entity type such as task, bug, or change-card, or use an entity key such as E07-F01-001, B034, or CC-042.
```

Do not guess.

## Notes

- This verb does not drive workflow execution. Use `/shark run <key>` for that.
- This verb does not call `shark next` unless the user explicitly asks what would
  be dispatched next. If they do, run `shark next <key> --preview` after the
  status and transition reads.
- Prefer the compact workflow view by default. Use `--all` only when the user
  asks for the expanded metadata view.
