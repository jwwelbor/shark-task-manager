---
epic_key: E13
title: Workflow-Aware Task Command System
description: Redesign task commands to work seamlessly with custom workflows and AI agent orchestration
---

# Workflow-Aware Task Command System

**Epic Key**: E13

---

## Goal

### Problem

The current shark task commands assume a hardcoded workflow (`todo → in_progress → ready_for_review → completed`) and don't work properly with custom workflows that have arbitrary phases and agent-specific transitions. AI agents and human users struggle with confusing command semantics:

- **`start`** assumes transition to `in_progress`, but custom workflows have `in_development`, `in_qa`, `in_code_review`, etc.
- **`complete`, `approve`, and `next-status`** overlap in functionality and don't clearly communicate phase completion vs. task completion
- **`reopen`** only makes sense from QA perspective, not a general backward transition
- Commands don't read workflow configuration to determine valid transitions
- No clear mental model for multi-agent handoffs (developer finishes → tech lead picks up)

This creates friction for the AI orchestrator system which needs to assign tasks to different agent types across workflow phases.

### Solution

Redesign task commands using a **phase-aware model** that reads workflow configuration and adapts to any custom workflow. Replace status-specific commands with intent-based commands:

- **`claim`** - Agent claims task in current phase (`ready_for_X → in_X`)
- **`finish`** - Agent completes their phase, advances to next (`in_X → ready_for_Y`)
- **`reject`** - Agent sends task backward for rework
- **Workflow-aware `next`** - Query tasks by phase and agent type

Commands will consult `.sharkconfig.json` workflow configuration to determine valid transitions, eliminating hardcoded assumptions. The mental model shifts from status-specific commands to **phase lifecycle** commands that work across any SDLC workflow.

### Impact

**For AI Agents:**
- Clear phase-based semantics that work with any custom workflow
- Self-documenting command intent (`claim → work → finish`)
- Automatic handoff to next agent type based on workflow config

**For Human Users:**
- Reduce from 25 to 18 commands (28% reduction)
- Eliminate confusion between `complete`, `approve`, and `next-status`
- Better alignment with SDLC terminology (claim, finish, reject)

**For AI Orchestrator:**
- Query tasks by phase (`ready_for_development`) and agent type
- Track work sessions (claim timestamp → finish timestamp)
- Detect stale work and reassign

**Expected Outcomes:**
- 100% compatibility with any custom workflow configuration
- 78% faster command discovery through categorization
- Zero workflow-assumption bugs in AI orchestrator
- Support for analytics CLI separation (`shark-analytics`)

---

## Business Value

**Rating**: High

This epic is critical for the AI orchestrator system and enables shark to support arbitrary SDLC workflows. Current hardcoded commands block adoption by teams with custom workflows (kanban, SAFe, custom) and create brittle AI agent code.

Strategic value:
- **Enables AI orchestrator**: Required for multi-agent task assignment and handoff
- **Workflow flexibility**: Allows shark to adapt to any team's process
- **Developer experience**: Clearer command semantics reduce learning curve
- **Future-proof**: New workflow phases don't require new commands

---

## Epic Components

This epic is documented across multiple interconnected files:

- **[User Personas](./personas.md)** - AI orchestrator agent, human PM/developer, workflow administrator
- **[User Journeys](./user-journeys.md)** - Multi-agent task handoff, custom workflow setup, PM work assignment
- **[Requirements](./requirements.md)** - Command specifications, workflow integration, migration requirements
- **[Success Metrics](./success-metrics.md)** - Compatibility rate, command usage patterns, migration success
- **[Scope Boundaries](./scope.md)** - Out of scope: analytics commands, workflow designer UI, auto-migration

---

## Quick Reference

**Primary Users**: AI Orchestrator Agent, Product Manager/Scrum Master, Software Developer

**Key Features**:
- Phase-aware `claim`, `finish`, `reject` commands replacing hardcoded workflow assumptions
- Workflow configuration reader for dynamic transition validation
- Feature-level `next` command for PM task assignment
- Command categorization (lifecycle, phase mgmt, work assignment, context, dependencies, history)
- Backward-compatible migration path with deprecation warnings

**Success Criteria**:
- 100% of workflow configurations work with new commands without modification
- AI orchestrator successfully assigns tasks across all phases
- 90% of users migrate to new commands within 2 releases

**Timeline**: Implementation in 3 phases over 6 weeks (non-breaking additions → deprecation → removal)

---

## Related Documentation

**Analysis & Design**:
- [Workflow-Aware Commands Proposal](../../../dev-artifacts/2026-01-10-task-command-ux-analysis/Workflow-Aware-Commands-Proposal.md)
- [CX Analysis: Command Organization](../../../dev-artifacts/2026-01-10-task-command-ux-analysis/CX-Analysis-I-2026-01-09-02.md)
- [AI Orchestrator Architecture](../../../.claude/docs/architecture/ai-agent-orchestrator-design.md)

**Idea Origin**:
- Idea I-2026-01-09-02: "simplify 'task' commands"

---

*Last Updated*: 2026-01-11
