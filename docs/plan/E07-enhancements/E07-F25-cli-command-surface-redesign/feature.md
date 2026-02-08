---
feature_key: E07-F25-cli-command-surface-redesign
epic_key: E07
title: CLI Command Surface Redesign
description: Hybrid verb-first/noun-first CLI redesign with smart dispatchers, top-level aliases, and unified create/delete commands
---

# CLI Command Surface Redesign

**Feature Key**: E07-F25-cli-command-surface-redesign

---

## Goal

### Problem
The Shark CLI has grown to 100+ command paths organized noun-first (`shark task create`, `shark epic list`). While this is well-organized, the most common daily operations require typing the entity noun even when the key format already encodes the entity type. Additionally, existing smart dispatchers (`shark list`, `shark get`) are underutilized because documentation promotes noun-first syntax.

### Solution
A hybrid approach that adds verb-first shortcuts for the daily workflow (next/start/done/block/unblock), unified dispatchers for create/delete, and promotes existing smart dispatchers in documentation -- all while keeping every existing command working exactly as-is.

### Impact
- Daily workflow commands shortened by 30-50% character count
- Zero breaking changes -- purely additive
- MCP tool surface reduced from 60+ narrow tools to 10 well-parameterized ones (future phase)

---

## Design Principles

1. **The key IS the entity type** -- auto-detect where possible
2. **Frequency-weighted ergonomics** -- daily loop (next/start/done) should be shortest
3. **Additive, never destructive** -- every existing command works forever
4. **One obvious way for common things** -- docs guide toward short form
5. **MCP tools mirror dispatchers** -- 10 rich tools, not 60 narrow ones (future)

---

## Implementation Phases

### Phase 1: Documentation (T-E07-F25-001)
Update docs to promote existing dispatchers as primary.

### Phase 2: Top-Level Aliases (T-E07-F25-002, T-E07-F25-003)
Add entity type detection helper and 5 lifecycle aliases.

### Phase 3: Unified Dispatchers (T-E07-F25-004, T-E07-F25-005)
Add `shark create <type>` and `shark delete <KEY>` dispatchers.

### Phase 4: Help & Polish (T-E07-F25-006, T-E07-F25-007)
Reorganize help groups and write comprehensive tests.

---

## Out of Scope

### Explicitly Excluded

1. **MCP Server Implementation**
   - **Why**: Separate feature, requires its own design
   - **Future**: Phase 4 in the broader CLI redesign plan

2. **Removing or deprecating existing noun-first commands**
   - **Why**: Zero breaking changes is a core principle
   - **Workaround**: Not needed -- both forms coexist permanently

3. **Tab completion for new commands**
   - **Why**: Cobra auto-generates completions, no extra work needed

---

## Tasks

| Key | Title | Priority | Dependencies |
|-----|-------|----------|-------------|
| T-E07-F25-001 | Promote existing dispatchers in docs | 5 | None |
| T-E07-F25-002 | Add entity type detection helper | 5 | None |
| T-E07-F25-003 | Add top-level aliases (next, start, done, block, unblock) | 7 | None |
| T-E07-F25-004 | Add unified create dispatcher | 5 | None |
| T-E07-F25-005 | Add unified delete dispatcher | 4 | T-E07-F25-002 |
| T-E07-F25-006 | Update help text and command groups | 4 | T-003, T-004, T-005 |
| T-E07-F25-007 | Write tests for new commands | 6 | T-002, T-003, T-004, T-005 |

---

*Last Updated*: 2026-02-08
