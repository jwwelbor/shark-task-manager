---
feature_key: E18-F04-bug-cli-commands
epic_key: E18
title: Bug CLI Commands
description: Implement the shark bug command group with create, get, list, update, delete, and triage subcommands as thin wrappers over BugService.
---

# Bug CLI Commands

**Feature Key**: E18-F04
**Execution Order**: 4 (depends on F02)

---

## Goal

### Problem

The BugService (F02) provides all bug business logic, but users have no way to invoke it. CLI commands are the primary entry point for all Shark operations.

### Solution

Implement `shark bug` as a new Cobra command group with the following subcommands, all following the thin-wrapper pattern (parse args, call service, format output):

- `shark bug create "<title>" [--severity=S] [--link=KEY]` -- Creates a bug
- `shark bug get B001 [--json] [--field NAME]` -- Retrieves bug details
- `shark bug list [--status=S] [--severity=S] [--link=KEY] [--json]` -- Lists bugs with filters
- `shark bug update B001 [--title="..."] [--severity=S]` -- Updates bug fields
- `shark bug delete B001 [--force]` -- Deletes a bug
- `shark bug triage B001 --severity=S [--assign=AGENT]` -- Triages a bug (sets severity, assigns, advances to triaged)
- `shark bug note add B001 --type=TYPE "content"` -- Adds a note
- `shark bug notes B001` -- Lists notes
- `shark bug context set/get/clear B001` -- Manages context fields
- `shark view B001` -- Renders the bug markdown file (via existing view command extension)

### Impact

- Developers can report bugs in under 30 seconds
- QA engineers can triage bugs with a single command
- All commands follow established CLI patterns for consistency
- `--json` output enables AI agent and CI/CD pipeline consumption

---

## Scope

### In Scope

1. **Command registration** -- `shark bug` parent command registered via `init()`. All subcommands nested under it.
2. **Create command** -- Parses title (positional), --severity, --link flags. Calls BugService.CreateBug. Outputs success message or JSON.
3. **Get command** -- Parses key (positional), --json, --field flags. Calls BugService.GetBug. Outputs formatted details or JSON.
4. **List command** -- Parses --status, --severity, --link, --json flags. Calls BugService.ListBugs. Outputs table or JSON.
5. **Update command** -- Parses key (positional), --title, --severity flags. Calls BugService.UpdateBug.
6. **Delete command** -- Parses key (positional), --force flag. Calls BugService.DeleteBug.
7. **Triage command** -- Parses key (positional), --severity, --assign flags. Calls BugService.TriageBug. This is a domain-specific compound operation.
8. **Note commands** -- Delegates to existing NoteService with entity_type="bug".
9. **Context commands** -- Delegates to existing ContextService with entity_type="bug".
10. **Tests** -- CLI tests with mocked BugService. No real database in CLI tests.

### Out of Scope

- Unified command integration (`shark get B001`) -- that is F06
- Dashboard display -- that is F07
- Bug promotion -- Could Have, deferred

---

## Requirements Traceability

| Epic Requirement | Coverage |
|-----------------|----------|
| REQ-F-001 (Bug Entity Creation) | `shark bug create` command |
| REQ-F-002 (Bug Severity Tracking) | --severity flag on create, list, triage |
| REQ-F-003 (Bug Entity Linking) | --link flag on create |
| REQ-F-005 (Bug Triage Command) | `shark bug triage` command |
| REQ-F-006 (Bug CRUD Commands) | create, get, list, update, delete subcommands |
| REQ-F-015 (Bug Notes and Context) | note and context subcommands |
| REQ-F-016 (Bug Markdown File Template) | `shark view B001` integration |
| REQ-F-018 (Bug List Filtering by Linked Entity) | --link flag on list |
| REQ-NF-005 (CLI Pattern Consistency) | All commands follow established patterns |

---

## Dependencies

- **F02 (Bug Entity Core)**: Requires BugService and all service methods.
- **Existing NoteService**: For bug notes.
- **Existing ContextService**: For bug context fields.

---

## Sprint Sizing

**Estimate**: 1 sprint (M complexity)

- Command file: Medium (many subcommands but each is a thin wrapper)
- Flag parsing: Small (follows established patterns)
- Tests: Medium (mock BugService, test each subcommand)
- Each subcommand is 15-25 lines following the parse-call-format pattern

---

*Last Updated*: 2026-03-02
