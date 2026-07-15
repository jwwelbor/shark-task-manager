# Use bounded council messages and artifacts

Use this contract for files below the configured council root, normally
`docs/council/`. It preserves scoped evidence across worker refreshes. It does
not grant workflow, status, or claim authority.

## Create an inbox message

Write one YAML file at `inbox/<recipient-role>/<message-id>.yaml`. Use stable,
lowercase, hyphenated IDs for the recipient role and message ID.

```yaml
message_id: msg-001
sender_role: architect
recipient_role: developer
root_key: E38
child_key: E38-F04
subject: Confirm the council boundary
requested_action: Implement the reviewed boundary.
urgency: normal
evidence:
  - docs/council/decisions/d-001.yaml
created_at: 2026-07-14T12:00:00Z
body: Use the bounded council artifact contract.
```

Include these fields:

| Field | Requirement |
|---|---|
| `message_id` | A unique stable ID for the inbox item. |
| `sender_role`, `recipient_role` | Stable roster IDs. |
| `root_key` | A valid Shark entity key. |
| `child_key` | An optional valid Shark entity key. |
| `subject` | A short description of the request. |
| `requested_action` or `question` | At least one is required. |
| `urgency` | `low`, `normal`, `high`, or `urgent`. |
| `evidence` | At least one relative, bounded evidence path. |
| `created_at` | An RFC 3339 timestamp. |
| `body` | Optional bounded context. Do not include a transcript. |

After acting, write the resulting durable artifact, then acknowledge or remove
the inbox message. Do not remove the only durable copy of a decision or
handoff. Repeating an acknowledgement after removal is a no-op.

## Create a durable artifact

Write decision, handoff, escalation, and resolution records as YAML. The
artifact type chooses the durable directory:

| Type | Directory |
|---|---|
| `decision` | `decisions/` |
| `handoff` | `handoffs/` |
| `escalation` | `escalations/` |
| `resolution` | `escalations/` |

```yaml
artifact_id: d-001
type: decision
status: open
roles:
  - architect
  - developer
root_key: E38
child_key: E38-F04
evidence:
  - docs/council/handoffs/h-001.yaml
created_at: 2026-07-14T12:00:00Z
updated_at: 2026-07-14T12:00:00Z
next_action: Developer implements the reviewed decision.
```

Every artifact requires `artifact_id`, `type`, `status`, `roles`, `root_key`,
`evidence`, `created_at`, `updated_at`, and `next_action`. An escalation also
requires `trigger` and `route`; use `council-review` when no project policy
selects another route. Reuse an artifact ID only with byte-equivalent content.
Changed content with an existing ID is a conflict, not an update.

## Keep content safe

Use only relative paths below the council root. The protocol rejects absolute
paths, `..` traversal, and symlink escapes. It also rejects credentials, access
tokens, rendered prompts, and unrestricted worker output. Return relative
artifact paths and structured metadata to downstream work; do not return
free-form role prose or transcript content.

## Resume work

Before acting after a refresh, load the scoped decisions, handoffs, unresolved
escalations, resolutions, and your inbox. Keep unresolved records in place so
the next worker receives the same bounded pointers.
