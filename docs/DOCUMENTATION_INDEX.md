# Documentation Index

Complete guide to all documentation for the Shark Task Manager project.

## Getting Started

Start here if you're new to the project:

1. [README](../README.md) - Project overview and quick start
2. [CLI Documentation](CLI_REFERENCE.md) - Complete CLI command reference
3. [Database Implementation](DATABASE_IMPLEMENTATION.md) - Database schema and design

## User Guides

### Epic & Feature Queries

Documentation for querying epics and features with automatic progress calculation:

| Document | Purpose | Best For |
|----------|---------|----------|
| [Epic & Feature Query Guide](EPIC_FEATURE_QUERIES.md) | Comprehensive documentation with detailed explanations of all commands, output formats, and error handling | First-time users, complete reference |
| [Quick Reference](EPIC_FEATURE_QUICK_REFERENCE.md) | Fast lookup for common commands and flags | Daily use, quick reminders |
| [Examples](EPIC_FEATURE_EXAMPLES.md) | Real-world scenarios and workflows with copy-paste examples | Learning by example, specific use cases |

**When to use which:**
- **Learning**: Start with the full guide, then use examples
- **Daily work**: Use quick reference for fast lookups
- **Problem solving**: Check examples for similar scenarios

### Task Management

Documentation for task lifecycle operations:

- Task creation and templating (coming soon)
- Task lifecycle operations (coming soon)
- File path management (coming soon)

## Technical Documentation

### Database

- [Database Implementation](DATABASE_IMPLEMENTATION.md) - Complete schema, indexes, and design decisions
- [internal/db/README.md](../internal/db/README.md) - Database package documentation

### Testing

- [Testing Guide](TESTING.md) - How to run tests and interpret results
- [Testing Guidelines](development/testing-guidelines.md) - Development testing best practices
- [Test Fixes Needed](development/test-fixes-needed.md) - Known testing issues

## Planning Documentation

### Project Structure

```
docs/
├── plan/                          # Project planning
│   ├── E04-task-mgmt-cli-core/   # Epic E04 planning docs
│   │   ├── epic.md               # Epic overview
│   │   ├── E04-F01-*/            # Feature F01 planning
│   │   │   └── tasks/                         # Task tracking
│   │   ├── E04-F02-*/            # Feature F02 planning
│   │   │   └── tasks/                         # Task tracking
│   │   ├── E04-F04-*/            # Feature F04 planning (Epic queries)
│   │   │   └── tasks/                         # Task tracking
│   │   └── ...
│   └── E05-task-mgmt-cli-capabilities/  # Epic E05 planning docs
│
└── templates/                     # Document templates
```

### Epic & Feature Planning

- [Epic E04: Task Management CLI Core](plan/E04-task-mgmt-cli-core/epic.md)
  - [E04-F01: Database Foundation](plan/E04-task-mgmt-cli-core/E04-F01-*/prd.md)
  - [E04-F02: CLI Infrastructure](plan/E04-task-mgmt-cli-core/E04-F02-*/prd.md)
  - [E04-F03: Task Lifecycle](plan/E04-task-mgmt-cli-core/E04-F03-task-lifecycle/prd.md)
  - [E04-F04: Epic & Feature Queries](plan/E04-task-mgmt-cli-core/E04-F04-epic-feature-queries/prd.md) ⭐
  - [E04-F05: File Path Management](plan/E04-task-mgmt-cli-core/E04-F05-*/prd.md)
  - [E04-F06: Task Creation](plan/E04-task-mgmt-cli-core/E04-F06-task-creation/prd.md)
  - [E04-F07: DB Initialization](plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/prd.md)

- [Epic E05: Task Management CLI Capabilities](plan/E05-task-mgmt-cli-capabilities/epic.md)
  - [E05-F01: Status Dashboard](plan/E05-task-mgmt-cli-capabilities/E05-F01-status-dashboard/prd.md)
  - [E05-F02: Dependency Management](plan/E05-task-mgmt-cli-capabilities/E05-F02-dependency-mgmt/prd.md)
  - [E05-F03: History & Audit](plan/E05-task-mgmt-cli-capabilities/E05-F03-history-audit/prd.md)

### Task Documentation

- [Master Task Index](tasks/MASTER-TASK-INDEX.md) - All tasks across all features
- [Task Index E04-F04](tasks/TASK-INDEX-E04-F04.md) - Epic & Feature Query tasks

## Future Enhancements

Documentation for planned features:

- [Output Formats](future-enhancements/output-formats.md) - Planned output format enhancements

## Documentation by Role

### For Developers

Essential documentation for developers working on the codebase:

1. [CLI Documentation](CLI.md) - Command reference
2. [Database Implementation](DATABASE_IMPLEMENTATION.md) - Database schema
3. [Testing Guide](TESTING.md) - How to test
4. [Epic & Feature Examples](EPIC_FEATURE_EXAMPLES.md) - Workflow examples

### For Product Managers

Documentation for planning and tracking progress:

1. [Epic & Feature Query Guide](EPIC_FEATURE_QUERIES.md) - How to track progress
2. [Epic & Feature Examples](EPIC_FEATURE_EXAMPLES.md) - Reporting scenarios
3. Task indexes in `tasks/` directory

### For AI Agents

Documentation optimized for AI agent consumption:

1. [Quick Reference](EPIC_FEATURE_QUICK_REFERENCE.md) - Fast command lookup
2. [Epic & Feature Examples](EPIC_FEATURE_EXAMPLES.md) - Agent workflow section
3. [CLI Documentation](CLI.md) - JSON output formats
4. PRD files in `plan/` directory - Detailed requirements

### For QA/Testing

Documentation for quality assurance:

1. [Testing Guide](TESTING.md) - How to run tests
2. [Testing Guidelines](development/testing-guidelines.md) - Testing best practices
3. PRD files - Acceptance criteria

## Documentation Standards

### File Naming Conventions

- `UPPERCASE.md` - Top-level documentation (README, CLI, DATABASE, etc.)
- `lowercase-with-dashes.md` - Guides and references
- `E##-F##-*` - Epic and feature specific documentation
- `T-E##-F##-###.md` - Task documentation

### Document Structure

All major guides follow this structure:

1. **Table of Contents** - Easy navigation
2. **Overview** - What this document covers
3. **Quick Start** - Get started immediately
4. **Detailed Sections** - In-depth information
5. **Examples** - Real-world usage
6. **Troubleshooting** - Common problems and solutions
7. **References** - Related documentation

### Cross-Referencing

Documentation uses relative links:
- `[Text](FILENAME.md)` - Same directory
- `[Text](../path/FILENAME.md)` - Parent directory
- `[Text](subdir/FILENAME.md)` - Subdirectory
- `[Text](FILENAME.md#section)` - Specific section

## Contributing to Documentation

### When to Create Documentation

Create documentation when:
- Implementing a new feature (user guide)
- Making architectural decisions (technical docs)
- Discovering common problems (troubleshooting)
- Creating reusable patterns (examples)

### Documentation Checklist

When creating new documentation:

- [ ] Clear title and purpose statement
- [ ] Table of contents (if >3 sections)
- [ ] Quick start or examples near the top
- [ ] Code examples with expected output
- [ ] Cross-references to related docs
- [ ] Troubleshooting section
- [ ] Version number and last updated date

### Documentation Review

Before marking documentation complete:

- [ ] All examples tested and working
- [ ] All cross-references valid
- [ ] Grammar and spelling checked
- [ ] Screenshots up-to-date (if applicable)
- [ ] Organized in logical sections
- [ ] Indexed in DOCUMENTATION_INDEX.md

## Finding What You Need

### By Task

| I want to... | Read this |
|--------------|-----------|
| List all epics | [Quick Reference](EPIC_FEATURE_QUICK_REFERENCE.md#list-all-epics) |
| Get epic details | [Epic & Feature Guide](EPIC_FEATURE_QUERIES.md#get-epic-details) |
| Track progress | [Epic & Feature Examples](EPIC_FEATURE_EXAMPLES.md#reporting-scenarios) |
| Debug progress calculation | [Troubleshooting](EPIC_FEATURE_QUERIES.md#troubleshooting) |
| Write a script | [Examples](EPIC_FEATURE_EXAMPLES.md#shell-script-integration) |
| Understand database schema | [Database Implementation](DATABASE_IMPLEMENTATION.md) |
| Run tests | [Testing Guide](TESTING.md) |
| Install the CLI | [README](../README.md#getting-started) |

### By Error Message

| Error | See |
|-------|-----|
| "Epic E99 does not exist" | [Epic & Feature Guide](EPIC_FEATURE_QUERIES.md#error-handling) |
| "Database error" | [Troubleshooting](EPIC_FEATURE_QUERIES.md#problem-database-error-when-running-commands) |
| "Invalid status" | [Quick Reference](EPIC_FEATURE_QUICK_REFERENCE.md#command-specific-flags) |
| Progress seems wrong | [Troubleshooting](EPIC_FEATURE_QUERIES.md#problem-progress-percentages-seem-incorrect) |

### By Feature

| Feature | Documentation |
|---------|---------------|
| Epic & Feature Queries | [Guide](EPIC_FEATURE_QUERIES.md), [Quick Ref](EPIC_FEATURE_QUICK_REFERENCE.md), [Examples](EPIC_FEATURE_EXAMPLES.md) |
| Database | [Database Implementation](DATABASE_IMPLEMENTATION.md) |
| CLI Framework | [CLI Documentation](CLI.md) |
| Testing | [Testing Guide](TESTING.md) |

## Documentation Maintenance

### Updating Documentation

Documentation should be updated when:
- Commands change (update CLI.md and relevant guides)
- New features added (create new guides)
- Bugs fixed that affect examples (update examples)
- User feedback indicates confusion (expand troubleshooting)

### Documentation Versioning

Major guides include version numbers:
- **Version 1.0.0** - Initial release
- **Last Updated** - Date of last significant update

Update version when:
- Major structural changes (1.x.x → 2.0.0)
- New sections added (x.1.x → x.2.0)
- Minor corrections (x.x.1 → x.x.2)

## Questions or Feedback

If you can't find what you're looking for:

1. Check the [CLI Documentation](CLI.md) first
2. Search for keywords in relevant guides
3. Review examples in [Epic & Feature Examples](EPIC_FEATURE_EXAMPLES.md)
4. Check the PRD files in `plan/` directory
5. File an issue with your question

---

**Index Version:** 1.0.0
**Last Updated:** 2025-12-15
**Maintained by:** Documentation Team
