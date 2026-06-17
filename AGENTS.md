# AGENTS.md

This repository keeps shared agent guidance in `CLAUDE.md` and `.claude/rules/`.

For any AI coding agent working in this repo:

1. Start with [CLAUDE.md](CLAUDE.md) for project overview, critical warnings, commands, and navigation.
2. Follow the path-specific rules under [.claude/rules/](.claude/rules/):
   - Architecture: [.claude/rules/architecture.md](.claude/rules/architecture.md)
   - Development workflow and quality gate: [.claude/rules/development-workflows.md](.claude/rules/development-workflows.md)
   - Go patterns: [.claude/rules/go/patterns.md](.claude/rules/go/patterns.md)
   - CLI commands: [.claude/rules/cli/commands.md](.claude/rules/cli/commands.md)
   - Services: [.claude/rules/services/service-design.md](.claude/rules/services/service-design.md)
   - Testing: [.claude/rules/testing/architecture.md](.claude/rules/testing/architecture.md)
3. Use [docs/architecture/coding-standards.md](docs/architecture/coding-standards.md) only for code-level standards.

Do not duplicate agent instructions here. Keep this file as a stable pointer so Codex and other agents can discover the same source of truth.
