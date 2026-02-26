# Shark CLI Reference

> **This file is a slim entry point.** The complete, authoritative CLI reference is at **[docs/cli-reference/README.md](cli-reference/README.md)**.

## Quick Start

```bash
# Quick Commands (task shortcuts)
shark next                                 # Get next available task
shark start E07-F01-001                    # Start task
shark done E07-F01-001 --notes="Done"      # Complete task

# Core Commands (auto-detect entity type)
shark get E07                              # Get epic
shark get E07-F01                          # Get feature
shark get E07-F01-001                      # Get task
shark list                                 # List epics
shark list E07                             # List features

# Status & Analytics
shark status                               # Project dashboard
shark progress E07                         # Epic progress
shark status history E07-F01-001           # Change history
```

## Full Documentation

| Topic | Link |
|-------|------|
| Quick Commands | [quick-commands.md](cli-reference/quick-commands.md) |
| Core Commands | [core-commands.md](cli-reference/core-commands.md) |
| Task Commands (26 subcmds) | [task-commands.md](cli-reference/task-commands.md) |
| Feature Commands (13 subcmds) | [feature-commands.md](cli-reference/feature-commands.md) |
| Epic Commands (14 subcmds) | [epic-commands.md](cli-reference/epic-commands.md) |
| Status & Analytics | [status-commands.md](cli-reference/status-commands.md) |
| Global Flags | [global-flags.md](cli-reference/global-flags.md) |
| Configuration | [configuration.md](cli-reference/configuration.md) |
| Best Practices | [best-practices.md](cli-reference/best-practices.md) |
| Error Messages | [error-messages.md](cli-reference/error-messages.md) |
| Workflow Configuration | [workflow-configuration.md](cli-reference/workflow-configuration.md) |

## Related

- [CLAUDE.md](../CLAUDE.md) - Development guidelines
- [README.md](../README.md) - Project introduction
- [Turso Setup](TURSO_QUICKSTART.md) - Cloud database
