# Turso Quick Start Guide

This guide will help you set up Shark Task Manager with [Turso](https://turso.tech), a cloud SQLite database, enabling access to your tasks from multiple machines.

## Prerequisites

1. **Install Shark CLI** (if not already installed):
   ```bash
   make install-shark
   ```

2. **Create a Turso Account**:
   - Sign up at [turso.tech](https://turso.tech)
   - Install the Turso CLI:
     ```bash
     curl -sSfL https://get.tur.so/install.sh | bash
     ```

## Step 1: Create a Turso Database

```bash
# Login to Turso
turso auth login

# Create a new database
turso db create shark-tasks

# Get the database URL
turso db show shark-tasks --url
# Output: libsql://shark-tasks-yourorg.turso.io

# Create an auth token
turso db tokens create shark-tasks
# Output: eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9...
```

## Step 2: Configure Shark for Turso

Run the cloud init command with your Turso credentials:

```bash
shark cloud init \
  --url="libsql://shark-tasks-yourorg.turso.io" \
  --auth-token="<your-token-here>" \
  --non-interactive
```

**Alternative: Store token in file (more secure)**:

```bash
# Save token to file
echo "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9..." > ~/.turso/shark-token
chmod 600 ~/.turso/shark-token

# Configure Shark to use token file
shark cloud init \
  --url="libsql://shark-tasks-yourorg.turso.io" \
  --auth-file="~/.turso/shark-token" \
  --non-interactive
```

**Alternative: Use environment variable**:

```bash
# Set environment variable
export TURSO_AUTH_TOKEN="eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9..."

# Configure Shark (will use env var)
shark cloud init \
  --url="libsql://shark-tasks-yourorg.turso.io" \
  --non-interactive
```

## Step 3: Verify Configuration

Check that cloud database is configured:

```bash
shark cloud status
```

Expected output:
```
Cloud Database Status
━━━━━━━━━━━━━━━━━━━━
Cloud database is CONFIGURED
Backend: turso
URL: libsql://shark-tasks-yourorg.turso.io
Auth token file: /path/to/token
```

## Step 4: Initialize Database Schema

If starting fresh, initialize the database:

```bash
shark init --non-interactive
```

This creates the necessary tables and indexes in your Turso database.

## Step 5: Start Using Shark

All shark commands now use your cloud database:

```bash
# Create an epic
shark epic create "Q1 2025 Goals"

# Create a feature
shark feature create E01 "User onboarding"

# Create a task
shark task create E01 F01 "Design welcome screen" --agent=frontend

# List tasks
shark task list

# Check task on another machine
# (After configuring cloud on that machine, tasks sync automatically)
```

## Multi-Machine Setup

To access your tasks from multiple machines:

1. **On each machine**, run `shark cloud init` with the same Turso URL and auth token
2. All machines will share the same database
3. Changes made on one machine are immediately visible on others

```bash
# Machine 1
shark task create E01 F01 "Implement login" --agent=backend
shark task start E01-F01-001

# Machine 2 (automatically sees the task)
shark task list E01 F01
# Shows: T-E01-F01-001 (in_development)
```

## Configuration Details

Your cloud configuration is stored in `.sharkconfig.json`:

```json
{
  "database": {
    "backend": "turso",
    "url": "libsql://shark-tasks-yourorg.turso.io",
    "auth_token_file": "/home/user/.turso/shark-token"
  }
}
```

**Security Notes**:
- **Recommended**: Store tokens in a separate file outside your project (e.g., `~/.turso/shark-token`)
- **Alternative**: Use environment variable `TURSO_AUTH_TOKEN`
- **Not recommended**: Storing token directly in `.sharkconfig.json` (less secure)

## Switching Back to Local Database

To switch back to local SQLite:

```bash
# Edit .sharkconfig.json
{
  "database": {
    "backend": "local",
    "url": "./shark-tasks.db"
  }
}
```

Or delete the `.sharkconfig.json` file to use defaults.

## Troubleshooting

### Error: "Failed to connect to database"
- Verify your Turso database URL is correct: `turso db show shark-tasks --url`
- Check your auth token is valid: `turso db tokens validate <token>`
- Ensure you have internet connection

### Error: "Auth token expired"
- Create a new token: `turso db tokens create shark-tasks`
- Update your config: `shark cloud init --url=... --auth-token=<new-token>`

### Database not syncing between machines
- Verify both machines use the same Turso URL
- Check that both machines have valid auth tokens
- Run `shark cloud status` on both machines to verify configuration

## Next Steps

- **Read the [Migration Guide](./TURSO_MIGRATION.md)** to migrate existing local data to Turso
- **See [CLI Reference](./CLI_REFERENCE.md)** for all cloud commands
- **Check [CLAUDE.md](../CLAUDE.md)** for AI agent instructions on using cloud databases

## Resources

- [Turso Documentation](https://docs.turso.tech)
- [Turso CLI Reference](https://docs.turso.tech/reference/turso-cli)
- [libSQL Client Go](https://github.com/tursodatabase/libsql-client-go)
