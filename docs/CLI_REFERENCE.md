# Shark CLI Reference

Complete command reference for the Shark Task Manager CLI.

## Quick Start

### Daily Workflow (Smart Dispatchers - Recommended)

For most daily operations, use the smart dispatchers which auto-detect entity type:

```bash
# List entities
shark list              # List all epics
shark list E07          # List features in epic E07
shark list E07 F01      # List tasks in feature E07-F01

# View details
shark get E07           # Get epic details
shark get E07-F01       # Get feature details
shark get E07-F01-001   # Get task details

# Check progress and status
shark status E07-F01    # See feature progress and task breakdown
shark history E07-F01-001  # See task change history

# Work with tasks
shark task next         # Get your next task to work on
shark task start E07-F01-001  # Start working on task
shark task complete E07-F01-001 --notes="Done with implementation"  # Mark for review
```

Smart dispatchers support:
- Case insensitive keys: `E07`, `e07`, all variants work
- `--json` flag: `shark get E07 --json` for machine-readable output
- Both numeric and slugged formats: `E07-F01` or `E07-F01-my-feature-name`

### Additional Resources

- **[Global Flags](cli-reference/global-flags.md)** - Flags available to all commands
- **[Key Formats](cli-reference/key-formats.md)** - Case insensitive keys, short formats, positional arguments

## Commands by Category

### Core Commands

- **[Initialization](cli-reference/initialization.md)** - `shark init` - Setup project infrastructure
- **[Epic Commands](cli-reference/epic-commands.md)** - Create, list, and manage epics
- **[Feature Commands](cli-reference/feature-commands.md)** - Create, list, and manage features
- **[Task Commands](cli-reference/task-commands.md)** - Create, list, and manage tasks
- **[Sync Commands](cli-reference/sync-commands.md)** - Synchronize files with database
- **[Configuration Commands](cli-reference/configuration.md)** - Manage configuration settings

### Advanced Topics

- **[Rejection Reasons](cli-reference/rejection-reasons.md)** - Document why tasks are rejected in review
- **[Orchestrator Actions](cli-reference/orchestrator-actions.md)** - API response format for AI orchestrators
- **[Enhanced JSON Fields](cli-reference/json-api-fields.md)** - Progress tracking, health indicators, rollups
- **[Error Messages](cli-reference/error-messages.md)** - Common errors and solutions
- **[Best Practices](cli-reference/best-practices.md)** - AI agent best practices, exit codes

## Configuration Files

- **[Workflow Configuration](cli-reference/workflow-config.md)** - Customize status flows, colors, phases
- **[Workflow Profiles](guides/workflow-profiles.md)** - Apply predefined workflow profiles (basic, advanced)
- **[Interactive Mode](cli-reference/interactive-mode.md)** - Configure interactive prompts

## Reference

- **[Dual Key Format](cli-reference/key-formats.md#dual-key-format-support)** - Numeric and slugged keys
- **[JSON Output Format](cli-reference/json-output.md)** - JSON response structures
- **[File Path Organization](cli-reference/file-paths.md)** - Custom file path support

## Related Documentation

- [CLAUDE.md](../CLAUDE.md) - Development guidelines and project overview
- [README.md](../README.md) - Project introduction and quick start
- [TURSO_QUICKSTART.md](TURSO_QUICKSTART.md) - Cloud database setup
- [TURSO_MIGRATION.md](TURSO_MIGRATION.md) - Migrate to cloud database
