---
epic_key: E17
title: CLI Simplification for AI Agents
description: Redesign the Shark CLI command structure to be optimized for AI agent users. Consolidate ~45 command paths to ~25 by unifying status commands, adding field extraction, batch operations, structured JSON errors, and environment-variable-driven output mode. Phased rollout with zero breaking changes.
---

# CLI Simplification for AI Agents

**Epic Key**: E17

---

## Goal

### Problem

The Shark CLI has ~45 command paths with inconsistent syntax, ambiguous status commands, and missing ergonomic features. AI agents (the primary CLI users) are forced to use defensive error handling (`2>/dev/null`), Python post-processing for field extraction, and manual bash for-loops as workarounds. Analysis of 231 real AI agent CLI interactions from the wormwoodGM project (collected 2026-02-16 to 2026-02-25) revealed:

- **75% of all interactions** are reading entities or changing status
- **36% of commands** use defensive `2>/dev/null` error suppression
- **15% of commands** pipe through Python to extract a single JSON field
- **5% of commands** are manual bash for-loops because no batch mode exists
- **4 different commands** exist for status changes (update --status, next-status, set-status, start/complete/approve), causing consistent agent confusion

### Solution

Redesign the CLI to invert the command hierarchy from noun-first (`shark task get`) to verb-first (`shark get <id>`), leveraging auto-detection of entity type from ID format. Add the missing features that agents need: `--field` extraction, batch status changes, structured JSON errors, and `SHARK_OUTPUT=json` environment variable. Deploy in three phases with zero breaking changes in Phase 1.

### Impact

- Command surface reduced from ~45 to ~25 paths
- Agent workflow commands per task lifecycle reduced from 8-10 (with fallbacks) to 5
- Python post-processing dependency eliminated
- Defensive error suppression patterns eliminated
- Manual for-loops for batch operations eliminated

---

## Business Value

**Rating**: High

AI agents are the primary CLI users (75%+ of all invocations). Every unnecessary command invocation, Python pipe, or error-suppression wrapper costs token budget and increases failure probability. This epic directly reduces agent friction on the two most common operations (entity retrieval and status changes), which together account for 75% of all CLI usage. The phased approach ensures zero disruption to existing workflows.

---

## Epic Components

This epic is documented across multiple interconnected files:

- **[Personas](personas.md)** - AI Development Agent, AI Orchestrator Agent, Human Developer
- **[User Journeys](user-journeys.md)** - Daily task workflow, batch transitions, project setup, status checks
- **[Requirements](requirements.md)** - 13 features across 3 phases with acceptance criteria
- **[Success Metrics](success-metrics.md)** - Measurable KPIs with baselines and targets
- **[Scope & Boundaries](scope.md)** - In-scope, out-of-scope, dependencies, risks, constraints

---

## Core Design Principles

1. **One obvious way to do everything** - No aliases, no "also available as" alternatives
2. **Verb-first, auto-detect entity** - ID format drives entity dispatch
3. **JSON-first for agents** - `SHARK_OUTPUT=json` env var, structured error output
4. **Consistent flags everywhere** - Same flags mean the same thing on every command
5. **Batch-native** - Accept multiple IDs wherever possible
6. **Idempotent where possible** - Setting status to current value succeeds silently

## Proposed Command Taxonomy

```
shark
  get <id> [--json] [--field <name>]         # auto-detect entity
  list [parent-id] [--status] [--json]       # auto-detect entity
  create epic|feature|task [parent] "title"  # entity type required
  update <id> [--title] [--priority] ...     # auto-detect entity
  delete <id> [--force]                      # auto-detect entity

  status set <id> <status> [--force] [--notes]     # direct status jump
  status advance <id> [--to <status>] [--notes]    # workflow-aware next
  status options <id>                                # valid transitions
  status history <id>                               # change log

  progress <id> [--json] [--field <name>]    # rollup, health, metrics
  next [--agent <type>] [--json]             # get next available task
  note <id> "text" [--type]                  # add note to entity

  admin init | config | cloud | workflow     # infrequent admin ops
```

---

## Quick Reference

**Primary Users**: AI Development Agents, AI Orchestrator Agents, Human Developers (see [personas.md](personas.md))

**Key Features**:
- Unified `status set` and `status advance` commands replacing 4+ scattered status commands
- `--field` flag for single-value extraction without Python
- Structured JSON error output with error codes
- `SHARK_OUTPUT=json` environment variable for session-wide JSON mode
- Batch status changes for multi-entity operations
- Consistent flag names across all commands

**Success Criteria** (see [success-metrics.md](success-metrics.md)):
- Command paths reduced from ~45 to ~25
- Agent commands per task lifecycle reduced from 8-10 to 5
- Python post-processing invocations reduced to 0%
- Defensive error suppression reduced from 36% to <5%
- Zero breaking changes in Phase 1
- 100% backward compatibility through hidden aliases for 2 releases

**Phased Delivery**:
- Phase 1 (Must-Have): F01-F05 -- add new commands alongside existing ones, zero breaking changes
- Phase 2 (Should-Have): F06-F08 -- promote new commands, add batch mode and progress command
- Phase 3 (Nice-to-Have): F09-F13 -- admin subgroup, deprecation warnings, unified update/delete

---

## Feature Summary

See [requirements.md](requirements.md) for the full requirements catalog with acceptance criteria.

### Phase 1: Add Without Removing (Must-Have)

| Feature | Description | Complexity | Pain Point Addressed |
|---------|-------------|------------|---------------------|
| F01 | `status` subcommand group (set/advance/options/history) | M | Status transition confusion |
| F02 | `--field` flag for targeted extraction | S | Python post-processing |
| F03 | Structured JSON error output | S | Defensive error handling |
| F04 | `SHARK_OUTPUT=json` environment variable | XS | Forgetting --json flag |
| F05 | Flag normalization (--order replaces --execution-order) | XS | Inconsistent flag names |

### Phase 2: Promote New Commands (Should-Have)

| Feature | Description | Complexity | Pain Point Addressed |
|---------|-------------|------------|---------------------|
| F06 | `progress` command (replaces overloaded `shark status <id>`) | M | Ambiguous status command |
| F07 | Batch mode for status changes | M | Manual for-loops |
| F08 | Unified `create` dispatcher | M | Inconsistent create syntax |

### Phase 3: Polish & Cleanup (Nice-to-Have)

| Feature | Description | Complexity |
|---------|-------------|------------|
| F09 | `admin` subgroup for config/init/cloud | S |
| F10 | Unified `note` command | S |
| F11 | Deprecation warnings on old commands | XS |
| F12 | `update` command (unified entity field updates) | M |
| F13 | `delete` command (unified entity deletion) | S |

---

## Open Questions

1. **Should `status set` for features/epics respect E16 multi-level workflows?** E16 introduces configurable workflows for epics and features. If E16 lands first, `status set` and `status advance` should use the E16 workflow engine. If not, these commands should use the current hardcoded epic/feature statuses and be designed so E16 integration is a drop-in enhancement.

2. **Should `--field` support nested field access?** Initial implementation supports top-level fields only (`--field status`). Nested access (`--field orchestrator_action.agent_type`) is deferred unless agent logs show demand.

3. **Should batch `status set --feature` require `--from` filter?** Without `--from`, it would change ALL tasks in a feature regardless of current status, which could be destructive. Requiring `--from` is safer but adds friction.

---

## Related Documents

- [CX Review: CLI for AI Agents](../cx-review-cli-ai-agents.md) - Full CX analysis with activity log evidence
- [Personas](personas.md) - Detailed persona descriptions
- [User Journeys](user-journeys.md) - Current vs. proposed journey maps
- [Requirements](requirements.md) - Full requirements catalog with acceptance criteria
- [Success Metrics](success-metrics.md) - Measurable KPIs with baselines
- [Scope & Boundaries](scope.md) - In-scope, out-of-scope, dependencies, risks
- [CLI Reference](../../CLI_REFERENCE.md) - Current CLI documentation
- [E15: Service Layer Architecture](../E15-service-layer-architecture-refactoring/epic.md) - Dependency for service layer
- [E16: Multi-Level Workflow](../E16-multi-level-workflow/epic.md) - Related epic/feature workflow system
- [Service Layer Migration Guide](../../guides/service-layer-migration.md) - E15 architecture patterns

---

*Last Updated*: 2026-02-25
