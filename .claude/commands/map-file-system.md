---
description: Generate comprehensive filesystem reference document
---

# Map File System

Generate a complete project structure reference document that enables AI agents and developers to quickly navigate the codebase.

## Usage

```bash
/map-file-system
```

Or use the shorter alias:
```bash
/map
```

## Implementation

This command invokes the research skill's filesystem mapping workflow.

**Skill**: `~/.claude/skills/research/workflows/map-filesystem.md`
**Context**: `~/.claude/skills/research/context/filesystem-mapping-guide.md`

## What This Does

The skill will:
1. Scan the project directory structure using fast `ls` commands
2. Read critical configuration files (package.json, tsconfig.json, etc.)
3. Analyze the organization and purpose of each directory
4. Generate a comprehensive file-system.md document at `/docs/architecture/file-system.md`

The output includes:
- Complete directory hierarchy
- Purpose and contents of each major directory
- Location of key files and components
- Where to add new features
- Navigation guidance for AI agents

## Output

Creates: `/docs/architecture/file-system.md`
