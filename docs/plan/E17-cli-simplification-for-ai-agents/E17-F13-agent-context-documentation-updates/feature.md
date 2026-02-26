---
feature_key: E17-F13-agent-context-documentation-updates
epic_key: E17
title: Agent Context Documentation Updates
description: Update CLAUDE.md and .claude/rules/ files that AI agents load as context. Critical because these are injected into every agent conversation.
---

# Agent Context Documentation Updates

**Feature Key**: E17-F13-agent-context-documentation-updates

---

## Goal

### Problem
CLAUDE.md and .claude/rules/ files contain pre-E17 command references (e.g., `shark history` which is now `shark status history`, Smart Dispatchers framing replaced by Quick Commands + Core Commands). These files are injected into every AI agent conversation, causing agents to use wrong commands and get errors.

### Solution
Update all agent context files to reflect E17 CLI structure. Replace stale command examples, update Quick Commands section, rewrite cli/commands.md rules file, and ensure all examples are verified against the actual binary.

### Impact
- AI agents get correct command guidance in every conversation
- Reduced agent errors from stale documentation
- CLAUDE.md Quick Commands section matches actual CLI

---

## Requirements

### Functional Requirements

1. **REQ-F-001**: Update CLAUDE.md Quick Commands section
2. **REQ-F-002**: Update .claude/rules/quickref.md
3. **REQ-F-003**: Rewrite .claude/rules/cli/commands.md
4. **REQ-F-004**: Update .claude/rules/cli/patterns.md
5. **REQ-F-005**: Update .claude/rules/development-workflows.md
6. **REQ-F-006**: Update docs/CLI_REFERENCE.md (redirect or rewrite)
7. **REQ-F-007**: Verify all updated docs against --help output

---

## Acceptance Criteria

- All command examples in CLAUDE.md verified working
- All command examples in .claude/rules/ files verified working
- No references to pre-E17 commands (shark history, Smart Dispatchers)
- Quick Commands section reflects actual CLI aliases

---

*Last Updated*: 2026-02-25
