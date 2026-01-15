---
paths: "{internal/db,internal/config}/**/*"
---

# Cloud Database Support (Turso)

This rule is loaded when working with database or configuration files.

Shark supports **two database backends**:
- **Local SQLite**: Default, file-based (`shark-tasks.db`)
- **Turso Cloud**: Cloud-hosted SQLite for multi-machine access

## Quick Setup

```bash
# 1. Create Turso database
turso db create shark-tasks
turso db show shark-tasks --url  # Get URL
turso db tokens create shark-tasks  # Get auth token

# 2. Configure Shark for Turso
shark cloud init \
  --url="libsql://shark-tasks-yourorg.turso.io" \
  --auth-token="<token>" \
  --non-interactive

# 3. Verify
shark cloud status

# 4. Initialize schema (if new database)
shark init --non-interactive
```

## Configuration

Cloud database is configured in `.sharkconfig.json`:

```json
{
  "database": {
    "backend": "turso",
    "url": "libsql://shark-tasks-yourorg.turso.io",
    "auth_token_file": "/home/user/.turso/shark-token"
  }
}
```

**Security Best Practices:**
- ✅ Store tokens in separate file: `--auth-file="~/.turso/token"`
- ✅ Use environment variable: `export TURSO_AUTH_TOKEN="..."`
- ❌ Don't commit tokens in `.sharkconfig.json` (add to `.gitignore`)

## Multi-Machine Usage

Once configured, all machines sharing the same Turso URL access the same database:

```bash
# Machine 1
shark task create E01 F01 "Implement API" --agent=backend
shark task start E01-F01-001

# Machine 2 (sees changes immediately)
shark task list E01 F01
# Output: T-E01-F01-001 (in_development)
```

## Cloud CLI Commands

```bash
# Initialize cloud database
shark cloud init --url=<turso-url> --auth-token=<token> --non-interactive

# Check configuration status
shark cloud status

# Check with JSON output
shark cloud status --json
```

## Switching Between Backends

**To cloud:**
```bash
shark cloud init --url="libsql://..." --auth-token="..." --non-interactive
```

**To local:**
```bash
# Edit .sharkconfig.json
{
  "database": {
    "backend": "local",
    "url": "./shark-tasks.db"
  }
}
```

## Migration

To migrate existing local data to Turso:

```bash
# 1. Export local database
sqlite3 shark-tasks.db .dump > shark-backup.sql

# 2. Configure Turso (see Quick Setup above)

# 3. Import to Turso
turso db shell shark-tasks < shark-backup.sql

# 4. Verify
shark task list
```

**See [TURSO_MIGRATION.md](../../docs/TURSO_MIGRATION.md) for detailed migration guide.**

## Troubleshooting

**Error: "Failed to connect to database"**
```bash
# Verify URL
turso db show shark-tasks --url

# Verify token
turso db tokens validate <token>

# Check config
shark cloud status
```

**Error: "Auth token expired"**
```bash
# Create new token
turso db tokens create shark-tasks

# Update config
shark cloud init --url="libsql://..." --auth-token="<new-token>" --non-interactive
```

## Documentation

- [Turso Quick Start](../../docs/TURSO_QUICKSTART.md) - Step-by-step setup guide
- [Migration Guide](../../docs/TURSO_MIGRATION.md) - Migrate from local to Turso
- [Turso Documentation](https://docs.turso.tech) - Official Turso docs
