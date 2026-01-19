# Configuration Commands

Commands for managing Shark configuration.

## `shark config set`

Set a configuration value.

**Usage:**
```bash
shark config set <key> <value>
```

**Examples:**

```bash
# Set default agent type
shark config set default_agent backend

# Set default priority
shark config set default_priority 5
```

---

## `shark config get`

Get a configuration value.

**Usage:**
```bash
shark config get <key>
```

**Examples:**

```bash
# Get default agent type
shark config get default_agent

# Get default priority
shark config get default_priority
```

## Configuration File

Configuration is stored in `.sharkconfig.json` at the project root.

**Example Configuration:**
```json
{
  "database": {
    "backend": "local",
    "url": "./shark-tasks.db"
  },
  "default_agent": "backend",
  "default_priority": 5,
  "interactive_mode": false
}
```

## Cloud Database Configuration

For cloud database setup, use the `shark cloud init` command instead of manually editing config.

See [Turso Quickstart](../TURSO_QUICKSTART.md) for cloud database configuration.

## Related Documentation

- [Interactive Mode](interactive-mode.md) - Configure interactive prompts
- [Workflow Configuration](workflow-config.md) - Customize workflow
- [Turso Quickstart](../TURSO_QUICKSTART.md) - Cloud database setup
