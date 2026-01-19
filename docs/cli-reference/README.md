# CLI Reference Documentation

This directory contains the modular Shark CLI reference documentation.

## Documentation Structure

### Core Commands
- [initialization.md](initialization.md) - `shark init` command
- [epic-commands.md](epic-commands.md) - Epic management commands
- [feature-commands.md](feature-commands.md) - Feature management commands (TODO)
- [task-commands.md](task-commands.md) - Task management quick reference
- [task-commands-full.md](task-commands-full.md) - Complete task commands (TODO)
- [sync-commands.md](sync-commands.md) - Sync commands (TODO)
- [configuration.md](configuration.md) - Configuration commands (TODO)

### Key Concepts
- [global-flags.md](global-flags.md) - Global flags available to all commands
- [key-formats.md](key-formats.md) - Key format improvements (case insensitive, short format, positional args)

### Advanced Topics
- [rejection-reasons.md](rejection-reasons.md) - Rejection reason workflow (TODO)
- [orchestrator-actions.md](orchestrator-actions.md) - Orchestrator API response format (TODO)
- [json-api-fields.md](json-api-fields.md) - Enhanced JSON response fields (TODO)

### Configuration
- [interactive-mode.md](interactive-mode.md) - Interactive mode configuration (TODO)
- [workflow-config.md](workflow-config.md) - Workflow configuration (TODO)

### Reference
- [error-messages.md](error-messages.md) - Common errors and solutions (TODO)
- [best-practices.md](best-practices.md) - AI agent best practices and exit codes (TODO)
- [json-output.md](json-output.md) - JSON output format reference (TODO)
- [file-paths.md](file-paths.md) - File path organization (TODO)

## Creating New Documentation

When adding new documentation:

1. Create the markdown file in this directory
2. Add it to the appropriate section in this README
3. Link to it from the main [CLI_REFERENCE.md](../CLI_REFERENCE.md)
4. Cross-link related documentation

## Documentation Guidelines

- Keep each file focused on a single topic
- Use clear headings and examples
- Cross-reference related documentation
- Include both human and machine-readable examples
- Document both table and JSON output formats
