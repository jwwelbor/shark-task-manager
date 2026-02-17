# Shark Configuration Reference

Complete reference for `.sharkconfig.json` - the central configuration file for Shark Task Manager.

## Overview

The `.sharkconfig.json` file is automatically created by `shark init` and contains:
- Database backend configuration
- User interface preferences
- Epic/feature/task workflow definitions
- Status metadata and orchestrator actions

**Location**: Project root directory (auto-detected by walking up from current directory)

## File Structure

```json
{
  "database": { },
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

## Top-Level Configuration

### Database Configuration

```json
{
  "database": {
    "backend": "$SHARK_DB_BACKEND",      // "local" or "turso"
    "url": "$SHARK_DB_URL",              // File path or libsql:// URL
    "auth_token_file": "$SHARK_AUTH_TOKEN_FILE"  // For Turso auth
  }
}
```

#### Local SQLite Backend

```json
{
  "database": {
    "backend": "local",
    "url": "./shark-tasks.db"
  }
}
```

**Fields:**
- `backend`: Must be `"local"`
- `url`: Relative or absolute path to SQLite database file

**Example:**
```bash
# Default local configuration (created by shark init)
{
  "database": {
    "backend": "local",
    "url": "./shark-tasks.db"
  }
}
```

#### Turso Cloud Backend

```json
{
  "database": {
    "backend": "turso",
    "url": "libsql://shark-tasks-yourorg.turso.io",
    "auth_token_file": "/home/user/.turso/shark-token"
  }
}
```

**Fields:**
- `backend`: Must be `"turso"`
- `url`: Turso database URL (format: `libsql://[database-name]-[org].turso.io`)
- `auth_token_file`: Path to file containing Turso auth token (recommended for security)

**Alternative:** Store token directly (not recommended for version control):
```json
{
  "database": {
    "backend": "turso",
    "url": "libsql://...",
    "auth_token": "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9..."
  }
}
```

**Environment Variables:**
You can use environment variables for sensitive values:
```bash
export SHARK_DB_BACKEND=turso
export SHARK_DB_URL=libsql://shark-tasks-yourorg.turso.io
export SHARK_AUTH_TOKEN_FILE=/home/user/.turso/shark-token
```

### UI Preferences

```json
{
  "color_enabled": true,
  "json_output": false,
  "interactive_mode": false,
  "require_rejection_reason": true
}
```

#### `color_enabled` (boolean)

Enable ANSI color output in terminal.

**Default:** `true`

**Usage:**
```bash
# Disable color via config
"color_enabled": false

# Or via flag
shark task list --no-color
```

#### `json_output` (boolean)

Default output mode for all commands.

**Default:** `false` (human-readable tables)

**Options:**
- `true`: All commands output JSON by default
- `false`: Human-readable output with tables and colors

**Usage:**
```bash
# Override per command
shark task list --json

# Or set globally
"json_output": true
```

#### `interactive_mode` (boolean)

Enable interactive prompts for confirmations and choices.

**Default:** `false` (non-interactive mode for AI agents)

**When enabled:**
- Prompts for confirmation on destructive operations
- Offers choices for ambiguous inputs
- Asks for missing required fields

**Usage:**
```bash
# Enable for human use
"interactive_mode": true

# Disable for AI agents (default)
"interactive_mode": false
```

#### `require_rejection_reason` (boolean)

Require rejection reason when moving task backwards in workflow.

**Default:** `true`

**When enabled:**
- `shark task update --status=in_development` (from ready_for_qa) requires `--notes`
- Rejection reasons stored in task history
- Helps track quality issues

### Default Values

```json
{
  "default_agent": null,
  "default_epic": null
}
```

#### `default_agent` (string | null)

Default agent type for `shark task next` and task creation.

**Default:** `null` (no default)

**Usage:**
```json
{
  "default_agent": "backend"
}
```

```bash
# Uses default_agent if set
shark task next

# Override per command
shark task next --agent=frontend
```

#### `default_epic` (string | null)

Default epic for task/feature creation when not specified.

**Default:** `null`

**Usage:**
```json
{
  "default_epic": "E07"
}
```

### Sync Metadata

```json
{
  "last_sync_time": "2026-01-16T23:22:45-06:00"
}
```

#### `last_sync_time` (RFC3339 timestamp)

Last successful database-to-filesystem sync.

**Auto-managed:** Updated by `shark sync` command

**Purpose:**
- Track sync recency
- Detect drift between database and files
- Used by tooling to determine if sync is needed

## Workflow Configuration

Each entity type (epic, feature, task) has its own workflow configuration block.

### Workflow Structure

```json
{
  "epic_workflow": {
    "version": "1.0",
    "status_flow": { },
    "status_metadata": { },
    "special_statuses": { }
  },
  "feature_workflow": {
    "version": "1.1",
    "status_flow": { },
    "status_metadata": { },
    "special_statuses": { }
  }
}
```

**Common Fields:**
- `version`: Workflow schema version
- `status_flow`: Valid status transitions
- `status_metadata`: Status configuration and orchestrator actions
- `special_statuses`: Lifecycle markers

See **[workflow-configuration.md](workflow-configuration.md)** for detailed workflow documentation.

### Task-Level Workflow (Legacy)

Task workflow can be defined at both the root level (legacy) and under `feature_workflow`:

```json
{
  "status_flow": { },         // Legacy: root-level task workflow
  "status_metadata": { },     // Legacy: root-level task status metadata
  "status_flow_version": "1.0",
  "special_statuses": { }
}
```

**Note:** Root-level task workflow is maintained for backward compatibility. New projects should use epic/feature-scoped workflows.

## Environment Variable Substitution

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

**Substitution Rules:**
- Variables use format `$VAR_NAME`
- Must be set before running shark commands
- No default values (command fails if variable is unset)
- No escaping (use literal `$` is not supported)

**Example:**
```bash
# Set environment variables
export SHARK_DB_BACKEND=turso
export SHARK_DB_URL=libsql://shark-tasks-myorg.turso.io
export SHARK_AUTH_TOKEN_FILE=/home/user/.turso/token

# Run shark (variables automatically substituted)
shark task list
```

## Configuration Management Commands

### Initialize Configuration

```bash
# Create default configuration
shark init --non-interactive

# Creates .sharkconfig.json with:
# - Local SQLite backend
# - Default UI preferences
# - Basic workflow definitions
```

### View Configuration

```bash
# Get specific value
shark config get database.backend
# Output: local

shark config get color_enabled
# Output: true

# View entire config
cat .sharkconfig.json | jq .
```

### Update Configuration

```bash
# Set value
shark config set color_enabled false
shark config set default_agent backend

# Update workflow (applies profile)
shark init update --workflow=advanced
```

## Configuration Validation

Shark validates configuration on load:

**Database Validation:**
- `backend` must be "local" or "turso"
- `url` must be valid (file path or libsql:// URL)
- For Turso: `auth_token` or `auth_token_file` required

**Workflow Validation:**
- All referenced statuses must be defined in metadata
- Status flow must be acyclic (no circular loops)
- Special statuses must reference valid statuses
- Orchestrator agent types must be valid

**UI Validation:**
- Boolean fields must be true/false
- String fields must be strings or null

## Example Configurations

### Minimal Local Configuration

```json
{
  "database": {
    "backend": "local",
    "url": "./shark-tasks.db"
  },
  "color_enabled": true,
  "json_output": false,
  "interactive_mode": false
}
```

### Turso Cloud Configuration

```json
{
  "database": {
    "backend": "turso",
    "url": "libsql://shark-tasks-myteam.turso.io",
    "auth_token_file": "/home/user/.turso/shark-token"
  },
  "color_enabled": true,
  "json_output": false,
  "interactive_mode": false,
  "require_rejection_reason": true,
  "default_agent": "backend"
}
```

### AI Agent Configuration

```json
{
  "database": {
    "backend": "local",
    "url": "./shark-tasks.db"
  },
  "color_enabled": false,
  "json_output": true,
  "interactive_mode": false,
  "require_rejection_reason": true
}
```

**Rationale:**
- `color_enabled: false` - AI doesn't need colors
- `json_output: true` - Easier to parse
- `interactive_mode: false` - No human to prompt

### Team Configuration with Defaults

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

## Configuration Best Practices

### Security

1. **Never commit auth tokens** to version control
   ```bash
   # Use token file instead
   "auth_token_file": "/home/user/.turso/token"

   # Or environment variable
   "auth_token_file": "$SHARK_AUTH_TOKEN_FILE"
   ```

2. **Add to .gitignore**
   ```gitignore
   # Ignore config with sensitive data
   .sharkconfig.json

   # Check in template instead
   !.sharkconfig.json.template
   ```

3. **Use environment variables** for CI/CD
   ```bash
   # In CI pipeline
   export SHARK_DB_BACKEND=turso
   export SHARK_DB_URL=$TURSO_DB_URL
   export SHARK_AUTH_TOKEN_FILE=/tmp/token
   ```

### Multi-Environment Setup

Use different configs for different environments:

```bash
# Development
.sharkconfig.dev.json

# Staging
.sharkconfig.staging.json

# Production
.sharkconfig.prod.json

# Use with --config flag
shark task list --config=.sharkconfig.prod.json
```

### Team Sharing

Share workflow definitions but keep local overrides:

```bash
# Team shared config (checked into git)
.sharkconfig.template.json

# Local developer config (gitignored)
.sharkconfig.json

# Copy and customize
cp .sharkconfig.template.json .sharkconfig.json
# Edit database.url for local development
```

## Troubleshooting

### "Failed to load config"

**Cause:** Invalid JSON syntax

**Solution:**
```bash
# Validate JSON
cat .sharkconfig.json | jq .

# Check for common issues:
# - Missing commas
# - Trailing commas
# - Unquoted strings
```

### "Invalid database backend"

**Cause:** `backend` field is not "local" or "turso"

**Solution:**
```json
{
  "database": {
    "backend": "local"  // Must be exactly "local" or "turso"
  }
}
```

### "Auth token required for Turso"

**Cause:** Turso backend without auth_token or auth_token_file

**Solution:**
```bash
# Option 1: Token file
"auth_token_file": "/path/to/token"

# Option 2: Environment variable
export TURSO_AUTH_TOKEN="eyJ..."
```

### "Status flow validation failed"

**Cause:** Invalid workflow configuration

**Solution:**
```bash
# Check status flow is acyclic
# Ensure all referenced statuses exist in metadata
# Verify special_statuses reference valid statuses

# Run validation
shark init update --dry-run
```

## Related Documentation

- **[workflow-configuration.md](workflow-configuration.md)** - Workflow system deep dive
- **[template-system.md](template-system.md)** - Template configuration
- **[CLI_REFERENCE.md](../CLI_REFERENCE.md)** - Command reference
