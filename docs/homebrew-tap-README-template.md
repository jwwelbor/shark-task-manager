# Homebrew Tap for Shark Task Manager

Official Homebrew tap for installing [Shark Task Manager](https://github.com/jwwelbor/shark-task-manager) on macOS.

## Installation

```bash
# Add the tap
brew tap jwwelbor/shark

# Install shark
brew install shark

# Verify installation
shark --version
```

## One-Line Installation

```bash
brew install jwwelbor/shark/shark
```

## What is Shark?

Shark Task Manager is a command-line tool for managing tasks, epics, and features in multi-agent software development projects. It provides a SQLite-backed database for tracking project state with commands optimized for both human developers and AI agents.

## Features

- Task management with status tracking
- Epic and feature organization
- Markdown-based task files
- SQLite persistence
- AI-agent optimized commands
- Rich terminal output

## Usage

```bash
# Create an epic
shark epic create "My Epic" "Epic description"

# List tasks
shark task list

# Start a task
shark task start T-E01-F01-001

# Complete a task
shark task complete T-E01-F01-001

# Get help
shark --help
```

## Updating

```bash
# Update tap
brew update

# Upgrade shark
brew upgrade shark
```

## Uninstallation

```bash
# Remove shark
brew uninstall shark

# Remove tap (optional)
brew untap jwwelbor/shark
```

## Supported Platforms

- macOS 11+ (Big Sur and later)
- Intel (x86_64) and Apple Silicon (ARM64)

## Other Installation Methods

- **Scoop (Windows)**: See [scoop-shark](https://github.com/jwwelbor/scoop-shark)
- **Manual Download**: See [Releases](https://github.com/jwwelbor/shark-task-manager/releases)

## Issues & Support

For bug reports and feature requests, please visit the [main repository](https://github.com/jwwelbor/shark-task-manager/issues).

## License

MIT License - see [LICENSE](https://github.com/jwwelbor/shark-task-manager/blob/main/LICENSE)
