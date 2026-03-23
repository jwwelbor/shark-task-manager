# API Reference

> Part of the Shark Task Manager Brownfield Analysis
<<<<<<< Updated upstream
> Generated: 2026-03-22
> Phase: 3 — Code Reference

## CLI API (Primary Interface)

### Core Commands (Entity Auto-Detection)

| Command | Args | Flags | Description |
|---------|------|-------|-------------|
| `shark get <key>` | Entity key | `--json`, `--field` | Get entity details |
| `shark list [epic] [feature]` | Optional scope | `--json`, `--status`, `--agent` | List entities |
| `shark create <type> [args]` | Type + args | `--order`, `--agent`, `--priority` | Create entity |
| `shark update <key>` | Entity key | `--title`, `--priority` | Update entity |
| `shark delete <key>` | Entity key | `--force` | Delete entity |
| `shark view <key>` | Entity key | | View markdown file |
| `shark search <query>` | Search text | `--type` | Cross-entity search |

### Status & Workflow Commands

| Command | Description | Output |
|---------|-------------|--------|
| `shark status` | Project dashboard | Summary table |
| `shark status <key>` | Entity status details | Detail view |
| `shark status set <key> <status>` | Set status directly | Confirmation |
| `shark status advance <key>` | Advance to next workflow status | New status |
| `shark status options <key>` | Show valid next statuses | Status list |
| `shark status history <key>` | Status change history | History table |

### Task Commands (19 subcommands)

| Command | Description |
|---------|-------------|
| `shark task create <epic> <feature> "title"` | Create task |
| `shark task get <key>` | Get task details |
| `shark task list [filters]` | List tasks |
| `shark task update <key>` | Update task fields |
| `shark task delete <key>` | Delete task |
| `shark task approve <key>` | Final approval |
| `shark task reopen <key>` | Reopen task |
| `shark task next-status <key>` | Advance workflow |
| `shark task set-status <key> <status>` | Set status |
| `shark task deps <key>` | Dependency tree |
| `shark task link <key1> <key2>` | Link entities |
| `shark task unlink <key1> <key2>` | Remove link |
| `shark task context set/get/clear <key>` | Context management |
| `shark task note add <key>` | Add note |
| `shark task notes <key>` | View notes |
| `shark task criteria <key>` | Acceptance criteria |
| `shark task resume <key>` | Resume with context |
| `shark task history <key>` | Change history |
| `shark task sessions <key>` | Session history |

### Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--json` | bool | false | Machine-readable JSON output |
| `--field <name>` | string | "" | Extract single field (implies --json) |
| `--no-color` | bool | false | Disable colored output |
| `--verbose` / `-v` | bool | false | Debug logging |
| `--db <path>` | string | auto | Override database path |
| `--config <path>` | string | auto | Override config file path |

### Exit Codes

| Code | Meaning | Example |
|------|---------|---------|
| 0 | Success | Command completed |
| 1 | Not found | Entity doesn't exist |
| 2 | Database error | Connection failure |
| 3 | Invalid state | Workflow violation |
| 4 | Field not found | --field with invalid name |

## HTTP API (Minimal — Reference Implementation)

### Current Endpoints (`cmd/server/main.go`)

| Method | Path | Description | Response |
|--------|------|-------------|----------|
| GET | `/` | Health message | `"Shark Task Manager API"` |
| GET | `/health` | DB connectivity check | `{"status": "ok"}` or error |

**Note**: The HTTP API is a minimal reference implementation. Full REST endpoints for task/feature/epic CRUD are planned but not yet implemented. The `cmd/server/services.go` file contains the complete `WireServices()` function ready for handler injection.

### Planned API Structure

Based on `cmd/server/services.go` `ServiceContainer`:

| Resource | Planned Endpoints |
|----------|-------------------|
| Tasks | `GET/POST /api/v1/tasks`, `GET/PATCH/DELETE /api/v1/tasks/{key}` |
| Features | `GET/POST /api/v1/features`, `GET/PATCH/DELETE /api/v1/features/{key}` |
| Epics | `GET/POST /api/v1/epics`, `GET/PATCH/DELETE /api/v1/epics/{key}` |
| Bugs | `GET/POST /api/v1/bugs`, `GET/PATCH/DELETE /api/v1/bugs/{key}` |
| Changes | `GET/POST /api/v1/changes`, `GET/PATCH/DELETE /api/v1/changes/{key}` |

## JSON Output Format

All CLI commands support `--json` for machine-readable output:

```json
// shark get E07-F01-001 --json
{
  "id": 42,
  "key": "T-E07-F01-001",
  "title": "Implement user authentication",
  "status": "in_progress",
  "agent_type": "backend",
  "priority": 8,
  "depends_on": "E07-F01-002",
  "execution_order": 1,
  "file_path": "docs/plan/E07/F01/tasks/T-E07-F01-001.md",
  "slug": "implement-user-authentication",
  "created_at": "2026-01-15T10:00:00Z",
  "updated_at": "2026-03-22T11:30:00Z"
}
```

```json
// shark status advance E07-F01-001 --json
{
  "entity_type": "task",
  "entity_key": "T-E07-F01-001",
  "from_status": "in_progress",
  "to_status": "ready_for_review",
  "transitioned": true,
  "is_backward": false,
  "is_forced": false
}
```

See also: [Program Structure](program-structure.md) | [Interfaces](interfaces.md) | [Data Models](data-models.md)
=======
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
>>>>>>> Stashed changes
