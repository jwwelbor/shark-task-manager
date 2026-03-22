# API Reference

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 3 — Code Reference

## CLI API

The primary interface is the `shark` CLI. All commands support `--json` for machine-readable output.

### Core Commands (Entity Auto-Detection)

| Command | Description | Key Format |
|---------|-------------|-----------|
| `shark get <key>` | Get entity details | E##, E##-F##, E##-F##-###, B###, CC-### |
| `shark list [epic] [feature]` | List entities | Positional args for scoping |
| `shark create <type> [args]` | Create entity | Type: epic, feature, task |
| `shark update <key> [flags]` | Update entity | Auto-detects type |
| `shark delete <key>` | Delete entity | Auto-detects type |
| `shark view <key>` | View markdown file | Auto-detects type |

### Status Commands

| Command | Description |
|---------|-------------|
| `shark status [key]` | Dashboard or entity status |
| `shark status set <key> <status> [--reason] [--force]` | Set status directly |
| `shark status advance <key>` | Advance to next workflow status |
| `shark status options <key>` | Show valid next statuses |
| `shark status history <key>` | View status change history |

### Entity-Specific Commands

**Task** (19 subcommands): create, get, list, update, delete, approve, reopen, next-status, set-status, deps, link, unlink, context, note, notes, criteria, resume, history, sessions

**Feature** (13 subcommands): create, get, list, update, delete, complete, next-status, set-status, context, note, notes, criteria, resume

**Epic** (14 subcommands): create, get, list, delete, update, complete, next-status, set-status, status, context, note, notes, resume

**Bug** (10 subcommands): create, get, list, update, delete, triage, note, notes, context

**Change** (10 subcommands): create, get, list, update, delete, approve, note, notes, context

**Idea** (6 subcommands): create, get, list, update, delete, promote

### Admin Commands

| Command | Description |
|---------|-------------|
| `shark admin init` | Initialize project |
| `shark admin init update --workflow=<profile>` | Apply workflow profile |
| `shark admin validate` | Validate project structure |
| `shark admin migrate slugs` | Backfill entity slugs |
| `shark admin cloud init/status` | Cloud database management |
| `shark admin config show/validate` | Configuration management |

## HTTP API

The HTTP API server (`cmd/server/`) provides a minimal implementation:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/health` | GET | Health check |
| `/api/schema` | GET | Database schema info |

**Note**: The HTTP API is development/internal use only. No authentication. The service wiring pattern is documented in `cmd/server/services.go` for future expansion.

## JSON Output Format

### Success Response
```json
{
  "key": "E07-F01-001",
  "title": "Task Title",
  "status": "in_progress",
  "priority": 5,
  "agent_type": "backend",
  "created_at": "2026-03-20T10:00:00Z",
  "updated_at": "2026-03-20T11:00:00Z"
}
```

### Error Response
```json
{
  "error": {
    "code": "not_found",
    "message": "task not found: E07-F01-999"
  }
}
```

### Field Extraction
```bash
shark get E07-F01-001 --field status
# Output: "in_progress"
```

---

See also: [Interfaces](interfaces.md) | [CLI Reference](../../docs/cli-reference/README.md)
