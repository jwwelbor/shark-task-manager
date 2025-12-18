# Scoop Bucket for Shark Task Manager

Official Scoop bucket for installing [Shark Task Manager](https://github.com/jwwelbor/shark-task-manager) on Windows.

## Installation

```powershell
# Add the bucket
scoop bucket add shark https://github.com/jwwelbor/scoop-shark

# Install shark
scoop install shark

# Verify installation
shark --version
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

```powershell
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

```powershell
# Update scoop
scoop update

# Upgrade shark
scoop update shark
```

## Uninstallation

```powershell
# Remove shark
scoop uninstall shark

# Remove bucket (optional)
scoop bucket rm shark
```

## Supported Platforms

- Windows 10/11
- x64 architecture

## Prerequisites

Scoop package manager is required. Install from [scoop.sh](https://scoop.sh):

```powershell
# Install Scoop (run in PowerShell)
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
Invoke-RestMethod -Uri https://get.scoop.sh | Invoke-Expression
```

## Other Installation Methods

- **Homebrew (macOS)**: See [homebrew-shark](https://github.com/jwwelbor/homebrew-shark)
- **Manual Download**: See [Releases](https://github.com/jwwelbor/shark-task-manager/releases)

## Issues & Support

For bug reports and feature requests, please visit the [main repository](https://github.com/jwwelbor/shark-task-manager/issues).

## License

MIT License - see [LICENSE](https://github.com/jwwelbor/shark-task-manager/blob/main/LICENSE)
