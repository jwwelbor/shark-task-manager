# Cloud Database Configuration Design

## Overview

This document describes how Shark Task Manager will support choosing between local SQLite and cloud Turso databases through configuration.

## Configuration Options

### 1. Config File (.sharkconfig.json)

Add new fields to the existing config structure:

```json
{
  "database": {
    "backend": "local",           // "local" or "turso"
    "url": null,                  // For turso: "libsql://your-db.turso.io"
    "auth_token": null            // For turso: auth token or path to token file
  },
  "color_enabled": true,
  "default_agent": null,
  // ... existing fields
}
```

### 2. Environment Variables

Support environment variable overrides:

```bash
# Database backend selection
export SHARK_DB_BACKEND="turso"

# Turso connection (libSQL URL format)
export SHARK_DB_URL="libsql://shark-tasks-jwwelbor.turso.io"

# Authentication
export TURSO_AUTH_TOKEN="eyJhbGc..."
```

### 3. Command-Line Flags

Extend existing `--db` flag to support both paths and URLs:

```bash
# Local SQLite (existing behavior)
shark task list --db=./shark-tasks.db

# Cloud Turso (new behavior)
shark task list --db=libsql://shark-tasks.turso.io

# Use environment variable
SHARK_DB_URL=libsql://... shark task list
```

## Priority Order

Configuration resolution follows this priority (highest to lowest):

1. **Command-line flag**: `--db=...`
2. **Environment variable**: `SHARK_DB_URL` or `SHARK_DB_BACKEND`
3. **Config file**: `.sharkconfig.json` `database.backend` and `database.url`
4. **Default**: Local SQLite at `./shark-tasks.db`

## URL Detection Logic

The system automatically detects backend type from the database URL:

```go
func DetectBackend(dbURL string) string {
    if strings.HasPrefix(dbURL, "libsql://") || strings.HasPrefix(dbURL, "https://") {
        return "turso"
    }
    return "local"
}
```

## Authentication Storage

For security, auth tokens should NOT be stored in `.sharkconfig.json`. Instead:

### Option 1: Environment Variable (Recommended)
```bash
export TURSO_AUTH_TOKEN="eyJhbGc..."
```

### Option 2: Secure File
```json
{
  "database": {
    "auth_token_file": "~/.shark/turso-token"
  }
}
```

The token file should be:
- Stored outside the project directory
- Have permissions `600` (read/write for owner only)
- Not committed to version control

### Option 3: System Keychain (Future Enhancement)
- macOS: Keychain Access
- Linux: GNOME Keyring / KWallet
- Windows: Credential Manager

## User Workflows

### Scenario 1: Default Local Usage (No Changes)
```bash
# Works exactly as before
shark task list
```

### Scenario 2: Switch to Cloud (Config File)
```bash
# One-time setup
shark cloud init
# Creates cloud database and updates .sharkconfig.json:
# {
#   "database": {
#     "backend": "turso",
#     "url": "libsql://shark-tasks-jwwelbor.turso.io"
#   }
# }

# Use cloud automatically
shark task list
```

### Scenario 3: Multi-Workstation Setup
```bash
# Workstation 1: Initial setup
shark cloud init
export TURSO_AUTH_TOKEN="..."  # Add to ~/.bashrc

# Workstation 2: Connect to existing cloud DB
shark cloud login
# Prompts for database URL and token
# Updates local .sharkconfig.json

# Both workstations now sync automatically
shark task list  # Fetches from cloud
```

### Scenario 4: Temporary Local/Cloud Switch
```bash
# Usually use cloud, but work offline with local copy
shark task list --db=./local-backup.db

# Or temporarily use cloud
shark task list --db=libsql://shark-tasks.turso.io
```

### Scenario 5: Offline Mode with Embedded Replica
```bash
# Configure embedded replica for offline support
shark cloud init --embedded-replica

# Creates local replica that syncs with cloud
# Works offline, syncs when online
shark task list  # Uses local replica (fast)
# Automatically syncs changes to cloud when connected
```

## Config Schema Updates

Update `internal/config/config.go`:

```go
type Config struct {
    // Existing fields
    LastSyncTime *time.Time `json:"last_sync_time,omitempty"`
    ColorEnabled *bool      `json:"color_enabled,omitempty"`

    // NEW: Database configuration
    Database *DatabaseConfig `json:"database,omitempty"`

    // ... other fields
}

type DatabaseConfig struct {
    Backend       string  `json:"backend,omitempty"`        // "local" or "turso"
    URL           string  `json:"url,omitempty"`            // Database URL or path
    AuthTokenFile string  `json:"auth_token_file,omitempty"` // Path to token file
    EmbeddedReplica bool  `json:"embedded_replica,omitempty"` // Enable offline mode
}
```

## Implementation Tasks

The following tasks in Epic E13 handle configuration:

1. **T-E13-F01-005**: Add configuration fields for database backend selection
   - Update `internal/config/config.go` with `DatabaseConfig` struct
   - Add config file migration for existing users
   - Support environment variable overrides

2. **T-E13-F03-001**: Implement 'shark cloud init' command
   - Interactive setup wizard
   - Prompts for Turso credentials
   - Updates `.sharkconfig.json` with cloud settings

3. **T-E13-F03-002**: Implement 'shark cloud login' command
   - Connect to existing cloud database
   - Save credentials securely
   - Test connection

4. **T-E13-F03-006**: Add cloud backend flag to existing commands
   - Update all commands to respect database config
   - Support `--db` flag with URLs
   - Implement config priority resolution

## Migration Path

For existing users switching from local to cloud:

```bash
# Step 1: Ensure local database is up to date
shark sync

# Step 2: Initialize cloud database
shark cloud init
# Prompts: "Export local database to cloud? (y/n)"

# Step 3: Export local data to cloud
shark cloud export
# Uploads all tasks, features, epics to Turso

# Step 4: Verify cloud database
shark task list --db=libsql://your-db.turso.io

# Step 5: Update config to use cloud by default
# (Done automatically by `shark cloud init`)

# Step 6: Keep local backup (optional)
cp shark-tasks.db shark-tasks-backup.db
```

## Backward Compatibility

- Default behavior unchanged (local SQLite)
- Existing `--db` flag still works for local paths
- All existing workflows continue to work
- Cloud features are **opt-in only**

## Security Considerations

1. **Never commit auth tokens** to version control
   - Add `.sharkconfig.json` with tokens to `.gitignore`
   - Use environment variables or separate token files

2. **Token file permissions**
   - Automatically set `chmod 600` on token files
   - Warn if permissions are too permissive

3. **HTTPS only**
   - Turso uses HTTPS/WSS for all connections
   - No plaintext credentials over network

4. **Local replica encryption** (future)
   - Consider encrypting local embedded replicas
   - Use system keychain for encryption keys

## Example .sharkconfig.json

### Local-only (default)
```json
{
  "color_enabled": true,
  "last_sync_time": "2026-01-04T12:00:00Z"
}
```

### Cloud with embedded replica
```json
{
  "color_enabled": true,
  "database": {
    "backend": "turso",
    "url": "libsql://shark-tasks-jwwelbor.turso.io",
    "auth_token_file": "~/.shark/turso-token",
    "embedded_replica": true
  },
  "last_sync_time": "2026-01-04T12:00:00Z"
}
```

## References

- Turso libSQL Go Driver: https://github.com/tursodatabase/libsql-client-go
- libSQL URL format: `libsql://<database>.<org>.turso.io?authToken=<token>`
- Embedded replicas: https://docs.turso.tech/sdk/go/reference#embedded-replicas
