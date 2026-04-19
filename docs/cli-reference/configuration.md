# Configuration Reference

Complete reference for Shark CLI configuration — both the `.sharkconfig.json` file and `shark config` commands.

## Configuration File (`.sharkconfig.json`)

The `.sharkconfig.json` file is automatically created by `shark init` and contains database, UI, and workflow settings.

**Location**: Project root directory (auto-detected by walking up from current directory).

### File Structure

```json
{
  "database": {
    "backend": "local",
    "url": "./shark-tasks.db"
  },
  "viewer": "glow",
  "color_enabled": true,
  "json_output": false,
  "interactive_mode": false,
  "require_rejection_reason": true,
  "template_directory": "shark-templates",
  "web": {
    "port": 7777
  },
  "default_agent": null,
  "default_epic": null,
  "last_sync_time": "2026-01-16T23:22:45-06:00",
  "epic_workflow": { },
  "feature_workflow": { },
  "status_flow": { },
  "status_metadata": { },
  "status_flow_version": "1.0",
  "special_statuses": { },
  "bug_workflow": { },
  "change_workflow": { }
}
```

The `bug_workflow` and `change_workflow` keys configure the workflows for the two standalone defect/change entity types. See [Bug Workflow Configuration](#bug-workflow-configuration) and [Change-Card Workflow Configuration](#change-card-workflow-configuration) below.

### Database Configuration

#### Local SQLite (Default)

```json
{
  "database": {
    "backend": "local",
    "url": "./shark-tasks.db"
  }
}
```

#### Turso Cloud

```json
{
  "database": {
    "backend": "turso",
    "url": "libsql://shark-tasks-yourorg.turso.io",
    "auth_token_file": "/home/user/.turso/shark-token"
  }
}
```

See [Turso Quickstart](../TURSO_QUICKSTART.md) for cloud setup.

### UI Preferences

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `color_enabled` | bool | `true` | Enable ANSI color output |
| `json_output` | bool | `false` | Default to JSON output for all commands |
| `interactive_mode` | bool | `false` | Enable interactive prompts |
| `require_rejection_reason` | bool | `true` | Require reason when rejecting tasks |
| `viewer` | string | `"cat"` | External viewer for `shark view` (e.g., `"glow"`, `"nano"`) |
| `template_directory` | string | `"shark-templates"` | Template directory path relative to project root for `.tmpl` files |

### Web Server Configuration

The `web` key configures the `shark web` dashboard server.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `web.port` | int | `0` (use 7777) | TCP port for `shark web`; `0` means auto-select from 7777–7790 |

**Port selection priority** (highest to lowest):
1. `--port` CLI flag — exact port, fails if busy
2. `web.port` in `.sharkconfig.json` — exact port, fails if busy
3. Built-in default: try 7777, then 7778–7790

**Example** — always start on port 9000:

```json
{
  "web": {
    "port": 9000
  }
}
```

You can still override it on the command line:

```bash
shark web --port 8888   # uses 8888 regardless of web.port in config
```

### Default Values

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `default_agent` | string/null | `null` | Default agent type for task creation and filtering |
| `default_epic` | string/null | `null` | Default epic for task/feature creation |

### Environment Variables

Shark supports environment variable substitution in config values:

```json
{
  "database": {
    "backend": "$SHARK_DB_BACKEND",
    "url": "$SHARK_DB_URL",
    "auth_token_file": "$SHARK_AUTH_TOKEN_FILE"
  }
}
```

| Variable | Description |
|----------|-------------|
| `SHARK_DB_BACKEND` | Database backend (`local` or `turso`) |
| `SHARK_DB_URL` | Database URL or file path |
| `SHARK_AUTH_TOKEN_FILE` | Path to Turso auth token file |
| `SHARK_OUTPUT` | Default output format (set to `json` for JSON) |

---

## Bug Workflow Configuration

The `bug_workflow` key configures the workflow for bug entities (`B###`). It follows the same structure as `epic_workflow` and `feature_workflow`.

### Structure

```json
{
  "bug_workflow": {
    "version": "1.0",
    "status_flow": {
      "reported": ["triaged", "duplicate", "wont_fix"],
      "triaged": ["in_fix", "wont_fix", "duplicate"],
      "in_fix": ["in_verification", "triaged"],
      "in_verification": ["resolved", "in_fix"],
      "resolved": [],
      "wont_fix": [],
      "duplicate": []
    },
    "status_metadata": {
      "reported": {
        "color": "red",
        "description": "Bug reported, awaiting triage",
        "phase": "planning",
        "progress_weight": 0.05,
        "responsibility": "agent",
        "agent_types": ["business-analyst"],
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "business-analyst",
          "skills": ["research", "shark"],
          "instruction_template": "bug/reported.tmpl"
        }
      },
      "triaged": {
        "color": "yellow",
        "description": "Triaged, ready for fix",
        "phase": "development",
        "progress_weight": 0.2,
        "responsibility": "agent",
        "agent_types": ["developer"],
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "developer",
          "skills": ["debugging", "implementation"],
          "instruction_template": "bug/triaged.tmpl"
        }
      },
      "in_fix": {
        "color": "blue",
        "description": "Fix in progress",
        "phase": "development",
        "progress_weight": 0.5,
        "responsibility": "agent",
        "agent_types": ["developer"],
        "orchestrator_action": {
          "action": "check_or_resume",
          "agent_type": "developer",
          "skills": ["debugging", "implementation"],
          "instruction_template": "bug/in_fix.tmpl"
        }
      },
      "in_verification": {
        "color": "cyan",
        "description": "Fix applied, awaiting verification",
        "phase": "review",
        "progress_weight": 0.8,
        "responsibility": "agent",
        "agent_types": ["qa"],
        "orchestrator_action": {
          "action": "check_or_resume",
          "agent_type": "qa",
          "skills": ["quality"],
          "instruction_template": "bug/in_verification.tmpl"
        }
      },
      "resolved": {
        "color": "green",
        "description": "Bug verified as fixed",
        "phase": "done",
        "progress_weight": 1.0,
        "responsibility": "none",
        "orchestrator_action": {
          "action": "archive",
          "instruction_template": "Bug {id} resolved."
        }
      },
      "wont_fix": {
        "color": "gray",
        "description": "Will not be fixed",
        "phase": "done",
        "progress_weight": 1.0,
        "responsibility": "none",
        "orchestrator_action": {
          "action": "archive",
          "instruction_template": "Bug {id} closed as wont_fix."
        }
      },
      "duplicate": {
        "color": "gray",
        "description": "Duplicate of another bug",
        "phase": "done",
        "progress_weight": 1.0,
        "responsibility": "none",
        "orchestrator_action": {
          "action": "archive",
          "instruction_template": "Bug {id} closed as duplicate."
        }
      }
    },
    "special_statuses": {
      "_start_": ["reported"],
      "_complete_": ["resolved", "wont_fix", "duplicate"]
    }
  }
}
```

### Template Variables for Bugs

Available in `instruction_template` for bug workflow:

| Variable | Description |
|----------|-------------|
| `{id}` | Bug key (e.g., `B001`) |
| `{title}` | Bug title |
| `{severity}` | Bug severity (`critical`, `high`, `medium`, `low`) |
| `{file_path}` | Path to bug markdown file |
| `{linked_entity_type}` | Type of linked entity (`epic`, `feature`, `task`) |
| `{linked_entity_key}` | Key of the linked entity |

---

## Change-Card Workflow Configuration

The `change_workflow` key configures the workflow for change-card entities (`CC-###`).

### Structure

```json
{
  "change_workflow": {
    "version": "1.0",
    "status_flow": {
      "proposed": ["approved", "declined"],
      "approved": ["in_progress", "declined"],
      "in_progress": ["completed", "approved"],
      "completed": [],
      "declined": []
    },
    "status_metadata": {
      "proposed": {
        "color": "yellow",
        "description": "Awaiting scope assessment and approval",
        "phase": "planning",
        "progress_weight": 0.1,
        "responsibility": "agent",
        "agent_types": ["business-analyst"],
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "business-analyst",
          "skills": ["assessment", "shark"],
          "instruction_template": "change/proposed.tmpl"
        }
      },
      "approved": {
        "color": "cyan",
        "description": "Approved, ready to implement",
        "phase": "development",
        "progress_weight": 0.2,
        "responsibility": "agent",
        "agent_types": ["developer"],
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "developer",
          "skills": ["implementation"],
          "instruction_template": "change/approved.tmpl"
        }
      },
      "in_progress": {
        "color": "blue",
        "description": "Implementation in progress",
        "phase": "development",
        "progress_weight": 0.6,
        "responsibility": "agent",
        "agent_types": ["developer"],
        "orchestrator_action": {
          "action": "check_or_resume",
          "agent_type": "developer",
          "skills": ["implementation"],
          "instruction_template": "change/in_progress.tmpl"
        }
      },
      "completed": {
        "color": "green",
        "description": "Change implemented and verified",
        "phase": "done",
        "progress_weight": 1.0,
        "responsibility": "none",
        "orchestrator_action": {
          "action": "archive",
          "instruction_template": "Change-card {id} completed."
        }
      },
      "declined": {
        "color": "red",
        "description": "Change request declined",
        "phase": "done",
        "progress_weight": 1.0,
        "responsibility": "none",
        "orchestrator_action": {
          "action": "archive",
          "instruction_template": "Change-card {id} declined."
        }
      }
    },
    "special_statuses": {
      "_start_": ["proposed"],
      "_complete_": ["completed", "declined"]
    }
  }
}
```

### Template Variables for Change-Cards

Available in `instruction_template` for change-card workflow:

| Variable | Description |
|----------|-------------|
| `{id}` | Change-card key (e.g., `CC-001`) |
| `{title}` | Change-card title |
| `{priority}` | Priority level (1–10) |
| `{requested_by}` | Name of requester |
| `{file_path}` | Path to change-card markdown file |

---

## Configuration Commands

### `shark config show`

Display current configuration including file location and all settings.

```bash
shark config show                    # Show full config
shark config show --patterns         # Show only pattern configuration
shark config show --json             # JSON output
```

### `shark config validate`

Check configuration file for errors and validate settings.

```bash
shark config validate
```

### `shark config validate-patterns`

Validate all regex patterns in `.sharkconfig.json`. Reports results grouped by entity type. Exits non-zero if errors found (CI-friendly).

```bash
shark config validate-patterns
shark config validate-patterns --json
```

### `shark config test-pattern`

Test a regex pattern against a test string. Shows captured groups and validates for entity type.

```bash
shark config test-pattern \
  --pattern="E(?P<number>\d{2})-(?P<slug>[a-z-]+)" \
  --test-string="E04-task-mgmt" \
  --type=epic

shark config test-pattern \
  --pattern="T-E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<number>\d{3})\.md" \
  --test-string="T-E04-F07-003.md" \
  --type=task
```

**Flags:**
- `--pattern` — Regex pattern to test
- `--test-string` — String to test against
- `--type` — Entity type: `epic`, `feature`, `task` (default: `epic`)

### `shark config add-pattern`

Add a pattern preset to configuration. Patterns are appended; duplicates are skipped.

```bash
shark config add-pattern --preset=special-epics
shark config add-pattern --preset=numeric-only
```

### `shark config list-presets`

List all available pattern presets with descriptions.

```bash
shark config list-presets
```

### `shark config show-preset`

Show full pattern structure for a specific preset in JSON format.

```bash
shark config show-preset standard
shark config show-preset special-epics
```

### `shark config get-format`

Get the configured generation format template for an entity type.

```bash
shark config get-format --type=task
shark config get-format --type=epic --json
```

### `shark config get-status-action`

Get the orchestrator action definition for a specific status. Useful for debugging workflow configuration.

```bash
shark config get-status-action ready_for_development
shark config get-status-action ready_for_development --task=T-E01-F03-002
shark config get-status-action blocked --json
```

---

## Example Configurations

### AI Agent

```json
{
  "database": { "backend": "local", "url": "./shark-tasks.db" },
  "color_enabled": false,
  "json_output": true,
  "interactive_mode": false,
  "require_rejection_reason": true,
  "template_directory": "shark-templates"
}
```

### Human Developer

```json
{
  "database": { "backend": "local", "url": "./shark-tasks.db" },
  "color_enabled": true,
  "json_output": false,
  "interactive_mode": true,
  "viewer": "glow",
  "template_directory": "shark-templates"
}
```

### Team with Cloud Database

```json
{
  "database": {
    "backend": "turso",
    "url": "libsql://shark-tasks-team.turso.io",
    "auth_token_file": "$HOME/.turso/team-token"
  },
  "color_enabled": true,
  "json_output": false,
  "interactive_mode": true,
  "require_rejection_reason": true,
  "template_directory": "shark-templates",
  "default_epic": "E07",
  "default_agent": "developer"
}
```

---

## Best Practices

1. **Never commit auth tokens** — Use `auth_token_file` or environment variables
2. **Add `.sharkconfig.json` to `.gitignore`** if it contains sensitive data; check in a `.sharkconfig.template.json` instead
3. **AI agents**: Set `json_output: true`, `color_enabled: false`, `interactive_mode: false`
4. **Multi-environment**: Use `--config` flag to switch between dev/staging/prod configs

## Observability

The optional `observability` key in `.sharkconfig.json` configures OpenTelemetry tracing, metrics, and structured logging (including optional file-based log output via `log_file` / `SHARK_LOG_FILE`). The subsystem is disabled by default with zero overhead when the `observability` key is absent.

For the complete field reference, environment variable overrides, and example configurations, see:

- [Observability Developer Guide](../guides/observability.md) — usage examples, troubleshooting, and file-destination setup
- [Observability Configuration Reference](../guides/observability-config-reference.md) — every field, every environment variable, and complete example configurations

## Related Documentation

- [Workflow Configuration](workflow-configuration.md) - Workflow system reference
- [Global Flags](global-flags.md) - CLI-level configuration flags
- [Setup Commands](setup-commands.md) - `shark init` and related setup
- [Turso Quickstart](../TURSO_QUICKSTART.md) - Cloud database setup
- [Observability Developer Guide](../guides/observability.md) - OTel tracing, metrics, and structured logging
- [Observability Configuration Reference](../guides/observability-config-reference.md) - Complete `observability.*` field reference
