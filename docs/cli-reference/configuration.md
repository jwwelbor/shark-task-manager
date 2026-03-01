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
  "default_agent": null,
  "default_epic": null,
  "last_sync_time": "2026-01-16T23:22:45-06:00",
  "epic_workflow": { },
  "feature_workflow": { },
  "status_flow": { },
  "status_metadata": { },
  "status_flow_version": "1.0",
  "special_statuses": { }
}
```

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
  "require_rejection_reason": true
}
```

### Human Developer

```json
{
  "database": { "backend": "local", "url": "./shark-tasks.db" },
  "color_enabled": true,
  "json_output": false,
  "interactive_mode": true,
  "viewer": "glow"
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

## Related Documentation

- [Workflow Configuration](workflow-configuration.md) - Workflow system reference
- [Global Flags](global-flags.md) - CLI-level configuration flags
- [Setup Commands](setup-commands.md) - `shark init` and related setup
- [Turso Quickstart](../TURSO_QUICKSTART.md) - Cloud database setup
