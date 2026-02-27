Shark is a command-line task management tool for AI Agents.

Examples:
  shark list E07                      List features in an epic
  shark get E07-F01-001               View task details
  shark create feature E07 "new feature" --description="important feature"
  shark update E07-F02 --filename="docs/plan/E07-important-epic/F02-new-feature/specs.md"
  shark status advance T-E07-F01-001  Advance to the next status in the workflow                      

Usage:
  shark [command] [--help]
  *Note: most commands have subcommands and flags. Use --help to view syntax.

Inspect:
  get          Get epic, feature, or task details
  list         List epics, features, or tasks
  search       Search tasks by various criteria
  view         View epic, feature, or task specification in external viewer

Manage:
  create       Create an epic, feature, or task
  update       Update an epic, feature, or task
  delete       Delete an epic, feature, or task
  notes        Search notes across all tasks
  context      Get or manage entity context data
  related-docs Manage related documents

Workflow:
  status       Change status for epic, feature, or task
  start        Start working on a task (alias for 'task start')
  next         Get next available task (alias for 'task next')
  block        Block a task (alias for 'task block')
  done         Mark task as complete (alias for 'task complete')
  unblock      Unblock a task (alias for 'task unblock')

Advanced:
  admin        Setup, configuration, and maintenance
  idea         Manage ideas
  analytics    Analyze work session patterns and metrics
  progress     Show progress, health indicators, and task breakdown
  completion   Generate the autocompletion script for the specified shell

Flags:
      --json            Output in JSON format (machine-readable)
      --field string    Extract a single field from JSON output (e.g., --field status)
  -h, --help            help for shark or shark commands
      --no-color        Disable colored output
  -v, --verbose         Enable verbose/debug output
      --version         version for shark
      --config string   Config file path (default: .sharkconfig.json)
      --db string       Database file path (default "shark-tasks.db")
