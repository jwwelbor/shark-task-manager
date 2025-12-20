# Feature: CLI Infrastructure & Framework

## Epic

- [Epic PRD](/home/jwwelbor/.claude/docs/plan/E04-task-mgmt-cli-core/epic.md)

## Goal

### Problem

The shark tool needs a robust command-line interface that both human developers and AI agents can use effectively. Developers need intuitive command structure with comprehensive help text, while agents require machine-readable JSON output and fast startup times. The CLI must support complex command hierarchies (`shark task list`, `shark epic get`, etc.), handle errors gracefully with actionable messages, and work consistently across Linux, macOS, and Windows. Without proper infrastructure, every feature would need to reimplement argument parsing, output formatting, error handling, and configuration management, leading to inconsistent behavior and maintenance nightmares.

### Solution

Build a Go CLI framework using Cobra (command routing) and pterm (terminal formatting) that provides the foundation for all shark commands. The framework implements a hierarchical command structure (`shark <resource> <action>`), global flags (--json, --no-color, --verbose), standardized output formatting (tables, JSON, colored terminal), and comprehensive error handling with exit codes. It includes a configuration system (.pmconfig.json) for user defaults, context management for database sessions, and utilities for progress indicators and input validation. All commands use consistent patterns for argument parsing, output rendering, and error reporting, ensuring that agents can reliably parse outputs and developers get helpful, actionable feedback.

### Impact

- **Developer Experience**: Intuitive command structure with tab completion, helpful error messages, and comprehensive help text reduces learning curve to <30 minutes
- **Agent Reliability**: Structured JSON output with guaranteed schema enables agents to parse command results with 100% accuracy
- **Performance**: <50ms CLI startup time ensures commands feel instant even for simple queries
- **Consistency**: All commands follow identical patterns for arguments, flags, output, and errors, reducing cognitive load
- **Foundation for Features**: Provides the infrastructure needed by E04-F03 (Task Operations), E04-F04 (Epic/Feature Queries), and all E05 features

## User Personas

### Primary Persona: Backend Developer (implementing CLI commands)

**Role**: Developer building task operations, epic queries, and other features using this framework
**Environment**: Go 1.21+, terminal, local development

**Key Characteristics**:
- Needs simple APIs to register new commands
- Wants automatic help text generation
- Requires consistent error handling across commands
- Benefits from reusable output formatting utilities

**Goals**:
- Add new commands without boilerplate (argument parsing, help text, error handling)
- Format output consistently (tables for humans, JSON for agents)
- Handle errors with appropriate exit codes
- Access database sessions through dependency injection

**Pain Points this Feature Addresses**:
- Manual argument parsing for every command
- Inconsistent error messages across commands
- Reinventing output formatting for each command
- Managing database connections manually

### Secondary Persona: AI Agent (Claude Code Agents)

**Role**: Executes CLI commands and parses output programmatically
**Environment**: Claude Code CLI, relies on fast, predictable command behavior

**Key Characteristics**:
- Requires machine-readable output (JSON)
- Needs fast command execution (<1 second)
- Cannot interact with prompts or confirmations
- Depends on consistent exit codes for error handling

**Goals**:
- Parse command output reliably (structured JSON)
- Detect errors via exit codes
- Run commands without interactive prompts
- Get clear error messages for debugging

**Pain Points this Feature Addresses**:
- Human-readable output is hard to parse
- Inconsistent output formats across commands
- Interactive prompts block automation
- Ambiguous error messages

### Tertiary Persona: Product Manager / Technical Lead

**Role**: Human developer using Shark CLI interactively
**Environment**: Terminal (bash/zsh), managing multiple projects

**Key Characteristics**:
- Expects modern CLI UX (colored output, progress bars)
- Uses tab completion for efficiency
- Needs helpful error messages with suggestions
- Values comprehensive help text and examples

**Goals**:
- Learn commands quickly through help text
- See formatted, readable output in terminal
- Get actionable error messages
- Configure defaults (epic, agent type) to reduce typing

**Pain Points this Feature Addresses**:
- Cryptic error messages requiring source code reading
- No help text or examples for complex commands
- Repetitive typing of common flags
- Plain text output that's hard to scan

## User Stories

### Must-Have User Stories

**Story 1: Define Command Structure**
- As a backend developer, I want to register commands using Cobra's command structure (`&cobra.Command{}`), so that I can add new commands without manual routing logic.

**Story 2: Parse Arguments and Flags**
- As a backend developer, I want automatic argument parsing with type validation, so that I don't manually check `sys.argv` or convert strings to integers.

**Story 3: Output JSON for Agents**
- As an AI agent, I want to add `--json` to any command and receive valid JSON output, so that I can parse results programmatically without regex.

**Story 4: Display Formatted Tables**
- As a developer, I want task lists displayed as formatted tables with columns for key, title, status, and priority, so that I can quickly scan results.

**Story 5: Handle Errors with Exit Codes**
- As a backend developer, I want exceptions automatically converted to appropriate exit codes (1=user error, 2=system error), so that agents can detect failures.

**Story 6: Generate Help Text**
- As a user, I want to run `shark --help`, `shark task --help`, or `shark task list --help` and see comprehensive documentation with examples, so that I can learn commands without reading source code.

**Story 7: Configure Defaults**
- As a developer, I want to set default epic or agent type in `.pmconfig.json`, so that I don't repeat `--epic=E01 --agent=frontend` on every command.

**Story 8: Provide Database Context**
- As a backend developer, I want command functions to access a database session via dependency injection or context, so that I don't manually create connections in every command.

### Should-Have User Stories

**Story 9: Enable Tab Completion**
- As a developer, I want to install shell completion for bash/zsh, so that I can tab-complete commands, subcommands, and flags.

**Story 10: Show Progress Indicators**
- As a developer, I want long-running commands (sync, bulk operations) to show progress spinners or bars, so that I know the command is working.

**Story 11: Colorize Output**
- As a developer, I want success messages in green, errors in red, and warnings in yellow, so that I can quickly identify command results.

**Story 12: Validate Config File**
- As a user, I want to run `shark config validate` to check `.pmconfig.json` for errors, so that I catch typos before they cause cryptic errors.

### Could-Have User Stories

**Story 13: Support Command Aliases**
- As a developer, I want to define aliases like `shark t` for `shark task`, so that I can save keystrokes for frequent commands.

**Story 14: Dry-Run Mode**
- As a developer, I want to add `--dry-run` to destructive commands, so that I can preview changes before committing.

## Requirements

### Functional Requirements

**Command Structure:**

1. The CLI must use a hierarchical command structure: `shark <resource> <action> [arguments] [flags]`
   - Resources: task, epic, feature, config, sync, backup
   - Actions: list, get, create, update, delete (CRUD operations)
   - Example: `shark task list --status=todo`

2. The CLI must be implemented using Cobra framework for command routing and argument parsing

3. All commands must be registered as Cobra commands (`cobra.Command` structs)

4. The entry point must be `shark` command installed as a console script

**Global Flags:**

5. The system must support global `--json` flag that converts any command output to valid JSON

6. The system must support global `--no-color` flag that disables all ANSI color codes

7. The system must support global `--verbose` flag that enables debug logging

8. The system must support global `--config` flag to specify custom config file path (default: `.pmconfig.json`)

9. Global flags must be available on all commands (inherited from root command group)

**Argument Parsing:**

10. The system must use Cobra flags for all command arguments: `cmd.Flags().StringVar()`, `cmd.Flags().BoolVar()`, etc.

11. Required arguments must raise clear errors if missing: "Error: Missing required argument: TASK_KEY"

12. Optional flags must have sensible defaults (e.g., `--priority=5`)

13. Enum-based flags must validate values: `--status=invalid` raises "Error: Invalid status. Must be one of: todo, in_progress, blocked, ready_for_review, completed"

14. Multiple values must be supported for filters: `--status=todo,in_progress` or `--status=todo --status=in_progress`

15. Boolean flags must use standard Go CLI syntax: `--force` (boolean flags, no negation needed)

**Output Formatting:**

16. The system must use pterm library for terminal formatting (tables, progress bars, colors)

17. Human-readable output must format task lists as tables with columns: Key | Title | Status | Priority | Agent

18. The system must truncate long text (descriptions, titles) to fit terminal width

19. The system must detect terminal width and adjust table formatting responsively

20. JSON output must be valid JSON with consistent schema for each command

21. JSON output must be pretty-printed by default, compact with `--json-compact` flag

22. Empty results must return `{"results": [], "count": 0}` in JSON mode, "No tasks found." in human mode

**Error Handling:**

23. The system must use consistent exit codes:
    - 0: Success
    - 1: User error (invalid arguments, validation failure)
    - 2: System error (database error, file not found)
    - 3: Validation error (constraint violation, invalid state transition)

24. User errors must display actionable messages: "Error: Task T-E01-F01-001 does not exist. Use 'shark task list' to see available tasks."

25. System errors must display error message and suggest running with `--verbose` for details

26. Validation errors must explain what rule was violated and how to fix it

27. All exceptions must be caught at the command level and converted to appropriate exit codes

28. Stack traces must only be shown in `--verbose` mode

**Configuration System:**

29. The system must support `.pmconfig.json` file in project root

30. Config file must support keys: `default_epic`, `default_agent`, `color_enabled`, `json_output`

31. The system must load config file on startup and merge with command-line flags (CLI flags take precedence)

32. Missing config file must not cause errors (use defaults)

33. Invalid JSON in config file must raise clear error: "Error: .pmconfig.json is not valid JSON: Unexpected token at line 5"

**Database Context Management:**

34. The system must provide database session via dependency injection or global context accessible to all commands

35. Database session must be created once per command invocation (not per subcommand)

36. Database session must automatically close after command completes (success or failure)

37. Database connection errors must be caught and converted to exit code 2

**Help Text:**

38. All commands must have docstrings that become help text

39. Help text must include:
    - Description of what the command does
    - List of arguments and options with descriptions
    - Examples showing common usage patterns
    - Related commands (see also)

40. Examples must be realistic and copy-paste-able

41. The system must support `--help` flag on all commands and command groups

**Logging:**

42. The system must log to stderr (not stdout) to avoid polluting command output

43. Default log level must be WARNING (only show important messages)

44. `--verbose` flag must set log level to DEBUG

45. Log format must include timestamp, level, and message: `[2025-12-14 10:30:45] DEBUG: Loading config from .pmconfig.json`

**Installation:**

46. The system must be buildable via `go build -o shark .` for development

47. The system must define dependencies in `go.mod`: cobra, pterm, gorm, golang-migrate

48. Cross-platform binaries must be built using: `GOOS=linux GOARCH=amd64 go build`

49. The system must specify Go version requirement: `>=1.21` in `go.mod`

### Non-Functional Requirements

**Performance:**

- CLI startup time (from command invocation to first code execution) must be <50ms (compiled binary)
- Help text rendering must be instant (<50ms)
- Table formatting for 100 rows must complete in <200ms
- JSON serialization for 1000 tasks must complete in <500ms

**Usability:**

- All error messages must be written in plain English (no technical jargon)
- Error messages must suggest next steps: "Run 'shark task list' to see available tasks"
- Help text must include examples for every command
- Commands must follow Unix conventions (lowercase, hyphen-separated: `task-list` not `taskList`)

**Accessibility:**

- Colorized output must be optional (`--no-color` flag)
- Unicode characters must only be used if terminal supports them (detect with pterm or isatty)
- Tables must render correctly in 80-column terminal width
- JSON output must be parseable by standard tools (`jq`, `json.Unmarshal()` in Go)

**Reliability:**

- Invalid arguments must never cause uncaught exceptions (show error message + exit 1)
- Database errors must never expose internal details (show user-friendly message)
- Config file errors must be specific: "Expected 'default_epic' to be a string, got integer"

**Maintainability:**

- All Cobra commands must be properly typed (Go's strong type system)
- All commands must have Short and Long descriptions for help text
- Shared utilities (table formatting, JSON output) must be in separate package
- Exit codes must be defined as constants (ExitSuccess = 0, ExitUserError = 1, etc.)

**Compatibility:**

- Must work on Linux, macOS, Windows
- Must work in bash, zsh, fish, PowerShell
- Must work in non-interactive mode (scripts, CI/CD)
- Must work with piped output: `shark task list --json | jq '.results[0]'`

## Acceptance Criteria

### Command Registration

**Given** I define a new command using `&cobra.Command{}`
**When** I run `shark --help`
**Then** the new command appears in the command list
**And** the command's Short description is shown

### Argument Parsing

**Given** I run `shark task get T-E01-F01-001`
**When** the command executes
**Then** the argument "T-E01-F01-001" is passed to the handler function as a string

**Given** I run `shark task list --status=todo --agent=frontend`
**When** the command executes
**Then** status="todo" and agent="frontend" are passed as parameters

**Given** I run `shark task list` without required argument `--epic`
**When** the command executes
**Then** an error is displayed: "Error: Missing required option '--epic'"
**And** exit code is 1

### JSON Output

**Given** I run `shark task list --json`
**When** the command executes
**Then** the output is valid JSON
**And** the JSON contains keys: `results`, `count`
**And** each result is a dictionary with task fields

**Given** I run `shark task get T-E01-F01-001 --json`
**When** the task exists
**Then** the output is a JSON object with task details
**And** `jq '.key'` successfully extracts the task key

### Table Formatting

**Given** I run `shark task list` (without --json)
**When** results are returned
**Then** output is a formatted table with columns: Key, Title, Status, Priority, Agent
**And** column widths adjust to terminal width
**And** long titles are truncated with "..." ellipsis

**Given** terminal width is 80 columns
**When** I run `shark task list`
**Then** the table fits within 80 columns
**And** no line wrapping occurs

### Error Handling

**Given** I run `shark task get INVALID-KEY`
**When** the task does not exist
**Then** error message is displayed: "Error: Task INVALID-KEY does not exist"
**And** exit code is 1 (user error)

**Given** the database file is corrupted
**When** I run any command
**Then** error message is displayed: "Error: Database error. Run with --verbose for details."
**And** exit code is 2 (system error)

**Given** I run `shark task start T-E01-F01-001` and task status is already "in_progress"
**When** the command executes
**Then** error message is displayed: "Error: Cannot start task with status 'in_progress'. Task must be 'todo'."
**And** exit code is 3 (validation error)

### Configuration Loading

**Given** `.pmconfig.json` contains `{"default_epic": "E01"}`
**When** I run `shark task list` (without --epic flag)
**Then** the command uses epic "E01" from config

**Given** `.pmconfig.json` contains `{"default_epic": "E01"}`
**When** I run `shark task list --epic=E02`
**Then** the command uses epic "E02" from CLI (overrides config)

**Given** `.pmconfig.json` contains invalid JSON
**When** I run any command
**Then** error message is displayed: "Error: .pmconfig.json is not valid JSON: ..."
**And** exit code is 1

### Help Text

**Given** I run `shark --help`
**When** the command executes
**Then** a list of all top-level commands is displayed
**And** each command has a brief description

**Given** I run `shark task list --help`
**When** the command executes
**Then** detailed help is displayed including:
- Command description
- Required arguments
- Optional flags
- Examples

### Colorized Output

**Given** I run `shark task list` in a color-supporting terminal
**When** results are displayed
**Then** status "completed" is shown in green
**And** status "blocked" is shown in red
**And** status "todo" is shown in yellow

**Given** I run `shark task list --no-color`
**When** results are displayed
**Then** no ANSI color codes are present in output
**And** plain text is displayed

### Database Context

**Given** a command handler function with access to the database session
**When** the command executes
**Then** the database session is available and valid
**And** I can query the database using the session
**And** the session is automatically closed after the command completes

### Exit Codes

**Given** I run a successful command
**When** the command completes
**Then** exit code is 0

**Given** I run a command with invalid arguments
**When** the command fails
**Then** exit code is 1

**Given** a database connection error occurs
**When** the command fails
**Then** exit code is 2

**Given** a validation constraint is violated
**When** the command fails
**Then** exit code is 3

## Out of Scope

### Explicitly NOT Included in This Feature

1. **Actual Task Commands** - This feature provides the CLI framework only. Commands like `shark task list`, `shark task start`, etc. are implemented in E04-F03 (Task Lifecycle Operations).

2. **Epic and Feature Commands** - Commands like `shark epic list`, `shark feature get` are in E04-F04 (Epic & Feature Queries).

3. **Business Logic** - Validation of task status transitions, dependency checking, and progress calculation are in other features.

4. **File Operations** - Moving task files between folders is in E04-F05 (Folder Management).

5. **Advanced Output** - Progress bars, dependency graphs, and dashboards are in E05-F01 (Status Dashboard).

6. **Shell Completion Implementation** - While the framework should support it, actually generating and installing completion scripts is deferred.

7. **Web Interface** - Only CLI is in scope. No HTTP server or web dashboard.

8. **Interactive Prompts** - No `input()` or interactive wizards. All parameters must be command-line arguments.

9. **Daemon Mode** - No background process or server mode. Commands are one-shot executions.

10. **Plugin System** - No support for third-party command plugins or extensions.
