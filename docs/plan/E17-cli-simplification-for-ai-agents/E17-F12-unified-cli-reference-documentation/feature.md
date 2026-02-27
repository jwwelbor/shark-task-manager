---
feature_key: E17-F12-unified-cli-reference-documentation
epic_key: E17
title: Unified CLI Reference Documentation
description: Merge docs/cli/ and docs/cli-reference/ into a single authoritative docs/cli-reference/ directory. Delete docs/cli/ after merging.
---

# Unified CLI Reference Documentation

**Feature Key**: E17-F12-unified-cli-reference-documentation

---

## Goal

### Problem
Two separate doc sets exist (`docs/cli/` and `docs/cli-reference/`) plus outdated files, none reflecting the E17 CLI restructure. AI agents (70% of CLI users) load these as context and get confused by stale command references, wrong flag names, and missing new commands.

### Solution
Merge both directories into a single authoritative `docs/cli-reference/` with files organized by E17 command categories. Delete `docs/cli/` after merging unique content. Every command, flag, and example verified against the actual CLI binary.

### Impact
- Single source of truth for CLI documentation
- All 121 command paths documented with verified examples
- AI agents get accurate context, reducing command errors

---

## Requirements

### Functional Requirements

1. **REQ-F-001**: Create README.md index with persona-aware quick start
2. **REQ-F-002**: Create quick-commands.md (next/start/done/block/unblock)
3. **REQ-F-003**: Create core-commands.md (get/list/create/delete/view)
4. **REQ-F-004**: Create status-commands.md (status + set/advance/options/history)
5. **REQ-F-005**: Create progress-analytics.md (progress, analytics)
6. **REQ-F-006**: Rewrite task-commands.md (merge 3 sources, all 26 subcommands)
7. **REQ-F-007**: Update feature-commands.md (merge, all 13 subcommands)
8. **REQ-F-008**: Update epic-commands.md (merge, all 13 subcommands)
9. **REQ-F-009**: Create idea-commands.md (6 subcommands)
10. **REQ-F-010**: Create context-commands.md (get/set/clear)
11. **REQ-F-011**: Create discovery-commands.md (search, notes, related-docs)
12. **REQ-F-012**: Create setup-commands.md (init, validate, migrate, cloud, workflow)
13. **REQ-F-013**: Update global-flags.md (add --field)
14. **REQ-F-014**: Update configuration.md (merge sources, add SHARK_OUTPUT)
15. **REQ-F-015**: Update workflow-configuration.md (merge sources)
16. **REQ-F-016**: Update best-practices.md (E17 patterns)
17. **REQ-F-017**: Update error-messages.md (structured JSON errors)
18. **REQ-F-018**: Delete docs/cli/ directory
19. **REQ-F-019**: Remove stale files (task-commands-full.md, MIGRATION_STATUS.md)

---

## Acceptance Criteria

- Single docs/cli-reference/ directory with all CLI documentation
- docs/cli/ deleted with no content loss (unique content merged)
- Every command example verified against `./bin/shark --help`
- No broken cross-references

---

*Last Updated*: 2026-02-25
